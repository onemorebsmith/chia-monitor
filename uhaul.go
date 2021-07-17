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
		go monitorFolder(k)
	}
}

func monitorFolder(path string) {
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
				moveFile(f.Name(), path)
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

func moveFile(fname string, path string) error {
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
			fits, err := checkSpace(sizeMb, o.path)
			if err != nil {
				return err
			}
			if !fits {
				continue // no space for plot
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
