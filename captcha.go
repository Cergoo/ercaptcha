// Copyright 2013 ercaptcha  All rights reserved.
// Use of this source code is governed by a BSD-style

package main

import (
	"encoding/json"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"time"
)

const (
	alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	border   = 5
)

type (
	conf struct {
		Port                string
		Urlget              string
		Urlcheck            string
		Nodeid              string
		Font                string
		Hight               int
		width               int
		Time_chan           uint8
		FontImg             []*image.RGBA
		Allowed_symbols     []int
		allowed_symbols_len int
		Count               int
	}

	I struct {
		c conf
	}
)

func NewCaptca() *I {
	var (
		point [2]int
	)

	this := I{}
	rand.Seed(time.Now().Unix())
	file, _ := ioutil.ReadFile("conf.json")
	e := json.Unmarshal(file, &this.c)

	if e != nil {
		panic(e)
	}
	this.c.allowed_symbols_len = len(this.c.Allowed_symbols)
	img, e := os.Open("fonts/" + this.c.Font + ".png")
	if e != nil {
		panic(e)
	}

	imgdecod, e := png.Decode(img)
	img.Close()
	if e != nil {
		panic(e)
	}
	this.c.Hight = imgdecod.Bounds().Dy()
	this.c.width = 35*this.c.Count + 40

	line := false
	linecount := 0
	// find first and last points lines
	for i := 0; i < imgdecod.Bounds().Dx(); i++ {
		_, _, _, a := imgdecod.At(i, 0).RGBA()
		if !line {
			if a > 0 {
				point[0] = i
				line = true
			}
		} else if a == 0 {
			point[1] = i
			line = false
			linecount++

			rect := image.Rect(0, 0, point[1]-point[0], this.c.Hight)

			m := image.NewRGBA(rect)
			draw.Draw(m, rect, imgdecod, image.Point{point[0], 1}, draw.Src)
			this.c.FontImg = append(this.c.FontImg, m)
		}
	}

	return &this
}

//wave random distortion
type wave struct {
	d1, d2, liney, delta int
	algo                 bool
}

// random init
func (t *wave) init() {
	t.liney = 20 + rand.Intn(10)

	if rand.Intn(2) == 0 {
		t.algo = true
		t.delta = 10
		t.d1 = 2 + rand.Intn(7)
		t.d2 = t.d1*5 + rand.Intn(5)
	} else {
		t.d1 = 30
		t.d2 = 75 + rand.Intn(5)
		t.delta = 10
		if rand.Intn(2) == 0 {
			t.d1 *= -1
			t.delta = 40
		}
	}
}

func (t *wave) wave(src, dst *image.RGBA, dx, dy int) {
	var (
		dstx float64
		a    uint32
		x, y int
	)

	if t.algo && rand.Intn(2) == 0 {
		t.d1 *= -1
	}

	for y = 0; y < src.Bounds().Dy(); y++ {
		if y == t.liney {
			dx += 1
			continue
		}
		for x = 0; x < src.Bounds().Dx(); x++ {
			if _, _, _, a = src.At(x, y).RGBA(); a > 0 {
				dstx = float64(dx+x+t.delta) + float64(t.d1)*math.Sin(6.28*float64(y)/float64(t.d2))
				dst.Set(int(dstx), y, src.At(x, y))
			}
		}
	}
}

func (this *I) Gen() (*image.RGBA, string) {
	var (
		r        int
		currentx int = 0
	)

	img := image.NewRGBA(image.Rect(0, 0, this.c.width, this.c.Hight))

	rezult := ""
	wave := new(wave)
	wave.init()
	for i := 0; i < this.c.Count; i++ {
		r = rand.Intn(this.c.allowed_symbols_len)
		rezult += string(alphabet[this.c.Allowed_symbols[r]])
		wave.wave(this.c.FontImg[this.c.Allowed_symbols[r]], img, currentx, 0)
		currentx += this.c.FontImg[this.c.Allowed_symbols[r]].Bounds().Dx()
	}

	return img, rezult
}
