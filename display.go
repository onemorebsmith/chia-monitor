package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/gdamore/tcell/v2"
)

type Display struct {
	Valid  bool
	screen tcell.Screen
}

func MakeDisplay() *Display {
	var err error
	d := &Display{}
	d.screen, err = tcell.NewScreen()
	if err != nil {
		d.Valid = false
		log.Println(err)
		return d
		//log.Fatal(err)
	}
	if err := d.screen.Init(); err != nil {
		d.Valid = false
		log.Println(err)
		return d
	}
	d.Valid = true

	defStyle := tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorWhite)
	d.screen.SetStyle(defStyle)

	return d
}

func (d *Display) Show() bool {
	if !d.Valid {
		log.Println("Display is not valid")
		return false
	}

	log.Println("Display is valid")
	d.screen.Show()
	return true
}

func (d *Display) BlockingPoll() {
	if !d.Valid {
		log.Println("Display is not valid")
		return
	}

	for {
		switch ev := d.screen.PollEvent().(type) {
		case *tcell.EventResize:
			log.Println("Resizing")
			d.screen.Sync()
			d.Refresh()
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape {
				log.Println("Exiting")
				d.screen.Fini()
				os.Exit(0)
			}
		}
	}
}

func (d *Display) Refresh() {
	if !d.Valid {
		return
	}
	d.screen.Clear()

	d.writeProcStates()
	d.writeMemStats()

	d.screen.Show()
}

const phaseProgress = 20.0
const tableProgess = 1.0

func (d *Display) writeMemStats() {
	w, _ := d.screen.Size()

	barsize := float32(w - 4)
	//style := tcell.StyleDefault.Foreground(tcell.ColorCadetBlue.TrueColor()).Background(tcell.ColorBlack)

	freestyle := tcell.StyleDefault.Foreground(tcell.ColorGreen.TrueColor()).Background(tcell.ColorBlack)
	usedstyle := tcell.StyleDefault.Foreground(tcell.ColorRed.TrueColor()).Background(tcell.ColorBlack)

	total := (float32)(meminfo["MemTotal"]) / 1024.0 / 1024.0
	free := (float32)(meminfo["MemAvailable"]) / 1024.0 / 1024.0
	freePct := free / total

	freeBars := int(barsize * freePct)
	fullBars := int(barsize * (1 - freePct))

	barIdx := 2
	barHeight := 4
	for i := 0; i < freeBars; i++ {
		d.screen.SetContent(barIdx, barHeight, '#', nil, freestyle)
		barIdx++
	}

	for i := 0; i < fullBars; i++ {
		d.screen.SetContent(barIdx, barHeight, '#', nil, usedstyle)
		barIdx++
	}
	emitStr(d.screen, w/2-9, barHeight-1, tcell.StyleDefault, fmt.Sprintf("[Mem: %f/%f GB] ", free, total))
}

func maxint(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func minint(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func scrolltext(str string, window int, offset int) string {
	ll := len(str)
	if ll > window {
		e := minint(window+accum, ll)
		e = maxint(ll, e)

		s := maxint(accum, 0)
		s = minint(s, ll-window)

		return str[s:e]
	}

	return str
}

var accum = 30

func (d *Display) writeProcStates() {
	// scroll variable
	sec := time.Now().Second()
	if sec > 30 {
		if accum > -15 {
			accum -= 1
		}
	} else {
		if accum < 15 {
			accum += 1
		}
	}

	w, h := d.screen.Size()
	style := tcell.StyleDefault.Foreground(tcell.ColorCadetBlue.TrueColor()).Background(tcell.ColorBlack)

	height := h / 2
	width := 2
	maxLength := w / 2

	plotters := make([]*PlotterState, 0, len(plotterStates))
	for _, state := range plotterStates {
		plotters = append(plotters, state)
	}

	sort.SliceStable(plotters, func(i, j int) bool {
		return plotters[i].Pid < plotters[j].Pid
	})

	for _, state := range plotters {

		phase := state.State["phase"]
		table := state.State["table"]
		bucket := state.State["bucket"]
		nBuckets := state.State["bucketSize"]
		stateStr := fmt.Sprintf("Phase %s Table %s Bucket %s/%s", phase, table, bucket, nBuckets)
		last := state.State["last"]
		ll := len(last)
		if ll > maxLength {
			last = scrolltext(last, maxLength, accum)
		}

		p, _ := strconv.ParseFloat(phase, 32)
		t, _ := strconv.ParseFloat(table, 32)

		progress := (p * phaseProgress) + (t * tableProgess)

		emitStr(d.screen, width, height, style, strconv.Itoa(state.Pid))
		emitStr(d.screen, width+20, height, style, fmt.Sprintf("%d %%", (int)(progress)))
		height += 1
		emitStr(d.screen, width, height, style, stateStr)
		height += 1
		emitStr(d.screen, width, height, style, last)
		height += 1

		emitStr(d.screen, width+5, height, style, "Phase")
		emitStr(d.screen, width+20, height, style, phase)
		height += 1
		emitStr(d.screen, width+5, height, style, "Table")
		emitStr(d.screen, width+20, height, style, table)
		height += 1
		emitStr(d.screen, width+5, height, style, "Bucket")
		emitStr(d.screen, width+20, height, style, bucket)
		height += 1

	}

	emitStr(d.screen, w/2-9, 1, tcell.StyleDefault, "Press ESC to exit.")
	emitStr(d.screen, w/2+9, 1, tcell.StyleDefault, fmt.Sprintf("Accum: %d", accum))
}
