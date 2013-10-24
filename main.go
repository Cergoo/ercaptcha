// Copyright 2013 ercaptcha  All rights reserved.
// Use of this source code is governed by a BSD-style

package main

import (
	i_cipher "code.google.com/p/go.crypto/twofish"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	cipherlen = 16
	lendb     = 100000000
)

type (
	rezult_json struct {
		img string
		id  string
	}

	/*	init vector type
	 *
	 */
	t_iv struct {
		id  byte
		val []byte
	}

	t_captchaserv struct {
		db               [lendb]uint8
		iv               [2]t_iv
		bcipher          *i_cipher.Cipher
		rwmu_chang_state sync.RWMutex
		counter          counter
		iochan           chan *rezult_json
	}
)

var (
	v_captcha     *I
	v_captchaserv *t_captchaserv
)

/*
save old encripter&decripter
create new encripter&decripter
create init vector
*/
func (t *t_captchaserv) chang_state() {
	for {
		<-time.Tick(time.Duration(v_captcha.c.Time_chan) * time.Hour)

		t.rwmu_chang_state.Lock()
		t.iv[1].id = t.iv[0].id
		copy(t.iv[1].val, t.iv[0].val)
		io.ReadFull(rand.Reader, t.iv[0].val)
		t.iv[0].id++
		t.rwmu_chang_state.Unlock()
		fmt.Print(t.iv[0].id, "\n")
	}
}

func new_captchaserv() *t_captchaserv {
	var (
		e error
	)
	defer func() {
		if e != nil {
			panic(e)
		}
	}()

	t := new(t_captchaserv)
	t.counter = counter{limit: lendb}
	t.iv[0].val = make([]byte, 16)
	t.iv[1].val = make([]byte, 16)
	t.iv[0].id = byte(0)
	//t.iv[1].id = byte(0)

	key := make([]byte, cipherlen)
	io.ReadFull(rand.Reader, key)
	t.bcipher, e = i_cipher.NewCipher(key)

	t.iochan = make(chan *rezult_json, 1000)

	go t.chang_state()

	go t.gofabric()
	go t.gofabric()

	return t
}

func (this *rezult_json) Write(p []byte) (n int, err error) {
	this.img += string(p)
	return 0, nil
}

func (t *t_captchaserv) gofabric() {
	for {
		rjson := new(rezult_json)
		img, rez := v_captcha.Gen()
		png.Encode(rjson, img)
		rjson.img = base64.StdEncoding.EncodeToString([]byte(rjson.img))

		rjson.id = "!" + rez + fmt.Sprint(t.counter.inc())
		rjson.id += strings.Repeat(" ", 16-len(rjson.id))
		//fmt.Println(rjson.id + "\n")

		t.iochan <- rjson
	}
}

func (t *t_captchaserv) get(w http.ResponseWriter, r *http.Request) {

	defer func() {
		if e := recover(); e != nil {
			fmt.Println(fmt.Errorf("%v", e))
		}
	}()
	if r.Method != "GET" {
		return
	}

	r.ParseForm()
	jsonp := r.FormValue("callback")

	rjson := <-t.iochan
	crypt := make([]byte, 17)
	fmt.Println(rjson.id)

	t.rwmu_chang_state.RLock()
	encrypter := cipher.NewCBCEncrypter(t.bcipher, t.iv[0].val)
	encrypter.CryptBlocks(crypt[1:], []byte(rjson.id))
	crypt[0] = t.iv[0].id
	rjson.id = v_captcha.c.Nodeid + base64.URLEncoding.EncodeToString(crypt)
	t.rwmu_chang_state.RUnlock()

	request := "{\"id\": \"" + rjson.id + "\", \n \"img\": \"" + rjson.img + "\"}"
	if jsonp != "" {
		request = jsonp + "(" + request + ");"
	}

	w.Header().Set("Content-Type", "text/javascript")
	w.Header().Set("Cache-Control", "no-store, no-cache")
	fmt.Fprint(w, request)

}

/*
rezult 0 - err
       1 - ok
	   2 - капча введена не правильно
	   3 - капча устарела
	   4 - попытка повторного использования капчи
*/
func (t *t_captchaserv) check(w http.ResponseWriter, r *http.Request) {

	defer func() {
		if e := recover(); e != nil {
			fmt.Println(fmt.Errorf("%v", e))
			fmt.Fprint(w, "0")
		}
	}()

	var (
		state_id uint8 = 0
	)

	decryptstr := make([]byte, 16)

	// /captcha/check/+1byte node id
	param := strings.SplitN(r.URL.Path[len(v_captcha.c.Urlcheck)+1:], "/", 2)
	encryptstr, _ := base64.URLEncoding.DecodeString(param[0])

	if t.iv[1].id == encryptstr[0] {
		state_id = 1
	} else if t.iv[0].id != encryptstr[0] {
		fmt.Fprint(w, "3")
		return
	}

	t.rwmu_chang_state.RLock()
	decrypter0 := cipher.NewCBCDecrypter(t.bcipher, t.iv[state_id].val)
	t.rwmu_chang_state.RUnlock()
	decrypter0.CryptBlocks(decryptstr, encryptstr[1:])

	if decryptstr[0] != 33 {
		fmt.Fprint(w, "3")
		return
	}

	id, err := strconv.ParseUint(strings.TrimSpace(string(decryptstr[5:])), 10, 32)
	if err != nil {
		fmt.Fprint(w, "0")
		return
	}

	if string(decryptstr[1:5]) == param[1] {
		if t.db[id] == 100 {
			fmt.Fprint(w, "4")
			return
		}
		fmt.Fprint(w, "1")
		t.db[id] = 100
		return
	}

	if t.db[id] > 1 {
		fmt.Fprint(w, "4")
		t.db[id] = 100
		return
	}
	fmt.Fprint(w, "2")
	t.db[id]++
}

func main() {
	runtime.GOMAXPROCS(2)
	v_captcha = NewCaptca()
	v_captchaserv = new_captchaserv()

	srvMux := http.NewServeMux()
	srvMux.HandleFunc(v_captcha.c.Urlget, v_captchaserv.get)
	srvMux.HandleFunc(v_captcha.c.Urlcheck, v_captchaserv.check)

	srv := &http.Server{
		Addr:           v_captcha.c.Port,
		Handler:        srvMux,
		ReadTimeout:    5 * time.Second,
		MaxHeaderBytes: 1 << 18,
	}
	log.Fatal(srv.ListenAndServe())
}
