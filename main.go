package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	qrweb "github.com/rphlo/qart/qrweb"
)


func carp(f http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Content-Type", "text/plain")
				fmt.Fprintf(w, "<pre>\npanic: %s\n\n%s\n", err, debug.Stack())
			}
		}()
		f.ServeHTTP(w, req)
	})
}

func Draw(w http.ResponseWriter, req *http.Request) {
	url := req.FormValue("url")
	if url == "" {
		url = "http://rphlo.com/"
	}
	if req.FormValue("upload") == "1" {
		upload(w, req, url)
		return
	}

	t0 := time.Now()
	img := req.FormValue("i")
	if !isImgName(img) {
		img = "pjw"
	}
	if req.FormValue("show") == "png" {
		i := loadSize(img, 48)
		var buf bytes.Buffer
		png.Encode(&buf, i)
		w.Write(buf.Bytes())
		return
	}
	if req.FormValue("flag") == "1" {
		flag(w, req, img)
		return
	}
	if req.FormValue("x") == "" {
		var data = struct {
			Name string
			URL  string
		}{
			Name: img,
			URL:  url,
		}
		runTemplate(w, "qr/main.html", &data)
		return
	}

	arg := func(s string) int { x, _ := strconv.Atoi(req.FormValue(s)); return x }
	targ := makeTarg(img, 17+4*arg("v")+arg("z"))

	m := &Image{
		Name:         img,
		Dx:           arg("x"),
		Dy:           arg("y"),
		URL:          req.FormValue("u"),
		Version:      arg("v"),
		Mask:         arg("m"),
		RandControl:  arg("r") > 0,
		Dither:       arg("i") > 0,
		OnlyDataBits: arg("d") > 0,
		SaveControl:  arg("c") > 0,
		Scale:        arg("scale"),
		Target:       targ,
		Seed:         int64(arg("s")),
		Rotation:     arg("o"),
		Size:         arg("z"),
	}
	if m.Version > 8 {
		m.Version = 8
	}

	if m.Scale == 0 {
		if arg("l") > 1 {
			m.Scale = 8
		} else {
			m.Scale = 4
		}
	}
	if m.Version >= 12 && m.Scale >= 4 {
		m.Scale /= 2
	}

	if arg("l") == 1 {
		data, err := json.Marshal(m)
		if err != nil {
			panic(err)
		}
		h := md5.New()
		h.Write(data)
		tag := fmt.Sprintf("%x", h.Sum(nil))[:16]
		if err := ioutil.WriteFile("qrsave/"+tag, data, 0666); err != nil {
			panic(err)
		}
		http.Redirect(w, req, "/qr/show/"+tag, http.StatusTemporaryRedirect)
		return
	}

	if err := m.Encode(req); err != nil {
		fmt.Fprintf(w, "%s\n", err)
		return
	}

	var dat []byte
	switch {
	case m.SaveControl:
		dat = m.Control
	default:
		dat = m.Code.PNG()
	}

	if arg("l") > 0 {
		w.Header().Set("Content-Type", "image/png")
		w.Write(dat)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, "<center><img src=\"data:image/png;base64,")
	io.WriteString(w, base64.StdEncoding.EncodeToString(dat))
	fmt.Fprint(w, "\" /><br>")
	fmt.Fprintf(w, "<form method=\"POST\" action=\"%s&l=2\"><input type=\"submit\" value=\"Save this QR code\"></form>\n", m.Link())
	fmt.Fprintf(w, "</center>\n")
	fmt.Fprintf(w, "<br><center><font size=-1>%v</font></center>\n", time.Now().Sub(t0))
}

func main() {
	http.Handle("/qr/frame", carp(qrweb.Frame))
	http.Handle("/qr/frames", carp(qrweb.Frames))
	http.Handle("/qr/mask", carp(qrweb.Mask))
	http.Handle("/qr/masks", carp(qrweb.Masks))
	http.Handle("/qr/arrow", carp(qrweb.Arrow))
	http.Handle("/qr/draw", carp(Draw))
	http.Handle("/qr/bitstable", carp(qrweb.BitsTable))
	http.Handle("/qr/encode", carp(qrweb.Encode))
	http.Handle("/qr/show/", carp(qrweb.Show))
	log.Fatal(http.ListenAndServe(":8082", nil))
}
