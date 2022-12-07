package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type Download struct {
	Url           string // Url de la descarga
	TargetPath    string // Ruta de destino
	TotalSections int    // Número de secciones en las que se dividirá la descarga (conexiones simultáneas al servidor)
}

func main() {

	startTime := time.Now() // Tiempo de inicio de la descarga

	d := Download{
		Url:           "https://www.dropbox.com/s/urmuwd87rlg5sdd/RESUMEN%20OCTAVOS%20DE%20FINAL%20MUNDIAL%20QATAR%202022.mp4?dl=1",
		TargetPath:    "final.mp4",
		TotalSections: 10,
	}

	err := d.Do()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("Descarga finalizada en %v seconds\n", time.Now().Sub(startTime).Seconds())
}

func (d Download) Do() error {

	fmt.Println("Conectando con el servidor...")

	r, err := d.getNewRequest("HEAD")

	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(r)

	if err != nil {
		return err
	}

	if resp.StatusCode > 299 {
		return errors.New(fmt.Sprintf("No se puede procesar la solicitud, la respuesta es: %v", resp.StatusCode))
	}

	size, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if (err != nil) || (size == 0) {
		return fmt.Errorf("No se puede obtener el tamaño del archivo")
	} else {
		fmt.Printf("Tamaño del archivo: %v bytes\n", size)
	}

	var sections = make([][2]int, d.TotalSections)

	sectionSize := size / d.TotalSections

	for i := range sections {
		if i == 0 {
			sections[i][0] = 0
		} else {
			sections[i][0] = sections[i-1][1] + 1
		}

		if i < d.TotalSections-1 {
			sections[i][1] = sections[i][0] + sectionSize
		} else {
			sections[i][1] = size - 1
		}
	}

	fmt.Println(sections)

	var wg sync.WaitGroup

	for i, s := range sections {
		wg.Add(1)

		// Se crea una copia de las variables i y s para que no se sobreescriban
		i := i
		s := s
		go func() {
			defer wg.Done()
			err := d.downloadSection(i, s)
			if err != nil {
				panic(err)
			}
		}()
	}

	wg.Wait()

	return nil
}

func (d Download) getNewRequest(method string) (*http.Request, error) {
	req, err := http.NewRequest(
		method,
		d.Url,
		nil,
	)

	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Silly Downloader Manager v001")
	return req, nil
}

func (d Download) downloadSection(i int, s [2]int) error {

	r, err := d.getNewRequest("GET")
	if err != nil {
		return err
	}
	r.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", s[0], s[1]))
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	fmt.Printf("Descargado %v bytes de la sección %v: %v\n", resp.Header.Get("Content-Length"), i, s)
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	e := ioutil.WriteFile(fmt.Sprintf("section-%v.tmp", i), b, os.ModePerm)
	if e != nil {
		return err
	}

	return nil
}
