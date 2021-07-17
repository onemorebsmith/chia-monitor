package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"time"
)

type outputDir struct {
	path string
	lock int32
}

var outdirs = []*outputDir{}

func startUhaul(cfg UhaulConfig) {
	for _, k := range cfg.FinalPaths {
		outdirs = append(outdirs, &outputDir{
			path: k,
			lock: 0,
		})
	}

	for _, k := range cfg.StagingPaths {
		log.Printf("[Uhaul] Starting monitoring %s", k)
		go monitorFolder(k, cfg)
	}
}

func pruneOld(before time.Time, path string) error {
	for _, o := range outdirs {
		info, err := ioutil.ReadDir(o.path)
		if err != nil {
			return fmt.Errorf("error checking directory %s, %+v", path, err)
		}

		for _, f := range info {
			if f.IsDir() {
				continue
			}

			if filepath.Ext(f.Name()) == ".plot" {
				if f.ModTime().Before(before) {
					fullPath := o.path + "/" + f.Name()
					log.Printf("[Uhaul] pruning old plot %s, create date %s", fullPath, f.ModTime().Format(time.ANSIC))
					os.Remove(fullPath)
					return nil
				}
			}
		}
	}

	return nil
}

func monitorFolder(path string, cfg UhaulConfig) {

	for {
		info, err := ioutil.ReadDir(path)
		if err != nil {
			log.Printf("[Uhaul] Error checking directory %s, %+v", path, err)
			break
		}

		for _, f := range info {
			if f.IsDir() {
				continue
			}

			if filepath.Ext(f.Name()) == ".plot" {
				moveFile(f.Name(), path, cfg.PruneDate)
			}
		}

		time.Sleep(30 * time.Second)
	}
}

func checkSpace(requiredMb int64, path string) (bool, error) {
	dfOutput, err := df(path, "-m")
	if err != nil {
		return false, err
	}

	return dfOutput.availableBlocks > int64(requiredMb), nil
}

func moveFile(fname string, path string, pruneDate time.Time) error {
	for _, o := range outdirs {
		if atomic.CompareAndSwapInt32(&o.lock, 0, 1) {
			defer func() { o.lock = 0 }() // reset at the end

			// acquired lock on dir
			now := time.Now()
			srcPath := path + "/" + fname
			destPath := o.path + "/" + fname

			fi, err := os.Stat(srcPath)
			if err != nil {
				return fmt.Errorf("error calling stat on file %s, err: %v", fname, err)
			}
			// get the size in mb
			sizeMb := fi.Size() / 1024 / 1024

			maxTries := 2
			for i := 0; i < maxTries; i++ { // check that the file fits, if not check if we have older file that can be pruned
				fits, err := checkSpace(sizeMb, o.path)
				if err != nil {
					return err
				}
				if fits {
					break
				}

				pruneOld(pruneDate, o.path)
			}

			log.Printf("[Uhaul] Moving '%s' => '%s'", srcPath, destPath)
			_, err = exec.Command("/usr/bin/rsync", "--remove-source-files", srcPath, destPath).Output()
			if err != nil {
				log.Printf("[Uhaul] Failed moving file '%s' => '%s': %+v", srcPath, destPath, err)
				continue
			}
			log.Printf("[Uhaul] Moved file '%s' => '%s in %f minutes", srcPath, destPath, time.Since(now).Minutes())
			return nil // finished
		}
	}

	return nil
}
