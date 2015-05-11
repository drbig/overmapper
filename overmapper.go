// See LICENSE.txt for licensing information.

package main

import (
	"bufio"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	VERSION = `1`
	MAPX    = 180
	MAPY    = 180
	LEVEL   = `L 10`
	SCALE   = 2

	xo = MAPX * SCALE
	yo = MAPY * SCALE
)

type Map struct {
	width, height int
	n, s, w, e    int
	maps          map[image.Point]string
}

var (
	ErrMultiChars = errors.New("multiple characters detected")
	ErrNotFound   = errors.New("no seen files found")

	rxpSeenFile = regexp.MustCompile(`\#(.*?)\.seen\.(-?\d+)\.(-?\d+)$`)
	CBG         = image.NewUniform(color.RGBA{0, 0, 0, 255})
	CFG         = image.NewUniform(color.RGBA{255, 255, 255, 255})
	CNOTE       = image.NewUniform(color.RGBA{0, 0, 255, 255})
	CGRID       = image.NewUniform(color.RGBA{255, 0, 0, 180})
	CORIGIN     = image.NewUniform(color.RGBA{0, 255, 0, 180})
)

func (m *Map) String() string {
	return fmt.Sprintf("Map %dx%d (%d)", m.width, m.height, len(m.maps))
}

func NewMap(path string) (*Map, error) {
	var id string

	m := &Map{maps: make(map[image.Point]string)}

	err := filepath.Walk(path, func(p string, i os.FileInfo, er error) error {
		if er != nil {
			return nil
		}

		if p != path && i.IsDir() {
			return filepath.SkipDir
		}

		if rm := rxpSeenFile.FindStringSubmatch(p); rm != nil {
			if id == "" {
				id = rm[1]
			}
			if id != rm[1] {
				return ErrMultiChars
			}

			x, err := strconv.Atoi(rm[2])
			if err != nil {
				return err
			}
			y, err := strconv.Atoi(rm[3])
			if err != nil {
				return err
			}

			m.maps[image.Point{x, y}] = p

			if x < m.w {
				m.w = x
			}
			if x > m.e {
				m.e = x
			}
			if y > m.n {
				m.n = y
			}
			if y < m.s {
				m.s = y
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(m.maps) == 0 {
		return nil, ErrNotFound
	}

	m.width = m.e - m.w + 1
	m.height = m.n - m.s + 1

	return m, nil
}

func (m *Map) Draw() (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, m.width*xo, m.height*yo))
	draw.Draw(img, img.Bounds(), CBG, image.ZP, draw.Src)

	iy := 0
	for y := m.s; y <= m.n; y += 1 {
		ix := 0
		for x := m.w; x <= m.e; x += 1 {
			fmt.Printf("At %dx%d - %dx%d\n", x, y, ix, iy)

			path, any := m.maps[image.Point{x, y}]
			if any {
				file, err := os.Open(path)
				if err != nil {
					return nil, err
				}

				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					line := scanner.Text()
					if line == LEVEL {
						break
					}
				}
				if err = scanner.Err(); err != nil {
					return nil, err
				}

				scanner.Scan()
				data := strings.NewReader(scanner.Text())

				var visited, length, position int
				for {
					n, err := fmt.Fscanf(data, "%d %d", &visited, &length)
					if err != nil && err != io.EOF {
						return nil, err
					}
					if n != 2 {
						break
					}

					if visited == 1 {
						x0 := position % MAPX
						y0 := position / MAPX
						x1 := (position + length - 1) % MAPX
						y1 := (position + length - 1) / MAPX

						if y0 == y1 {
							// simple same-line case
							draw_rect(img, CFG, draw.Src, ix, iy, x0, y0, x1, y1)
						} else {
							// multi-line case, try to make big blocks if possible
							draw_rect(img, CFG, draw.Src, ix, iy, x0, y0, MAPX-1, y0)
							if y0+1 != y1 {
								draw_rect(img, CFG, draw.Src, ix, iy, 0, y0+1, MAPX-1, y1-1)
							}
							draw_rect(img, CFG, draw.Src, ix, iy, 0, y1, x1, y1)
						}
					}
					position += length
				}

				scanner.Scan() // E 10
				scanner.Scan() // 0 32400

				var nx, ny int
				for scanner.Scan() {
					data = strings.NewReader(scanner.Text())
					n, _ := fmt.Fscanf(data, "N %d %d", &nx, &ny)
					if n != 2 {
						break
					}

					draw_rect(img, CNOTE, draw.Over, ix, iy, nx, ny, nx, ny)

					scanner.Scan() // skip note text
				}

				file.Close()
			}

			// draw grid/origin
			var c image.Image
			if x == 0 && y == 0 {
				c = CORIGIN
			} else {
				c = CGRID
			}

			draw_rect(img, c, draw.Over, ix, iy, 0, 0, MAPX-1, 0)           // top-edge
			draw_rect(img, c, draw.Over, ix, iy, 0, MAPY-1, MAPX-1, MAPY-1) // bottom-edge
			draw_rect(img, c, draw.Over, ix, iy, 0, 0, 0, MAPY-1)           // left-edge
			draw_rect(img, c, draw.Over, ix, iy, MAPX-1, 0, MAPX-1, MAPY-1) // right-edge

			ix += 1
		}
		iy += 1
	}

	return img, nil
}

func draw_rect(img draw.Image, color image.Image, mode draw.Op, ix, iy, x0, y0, x1, y1 int) {
	x0 = (ix * xo) + (x0 * SCALE)
	y0 = (iy * yo) + (y0 * SCALE)
	x1 = (ix * xo) + ((x1 + 1) * SCALE)
	y1 = (iy * yo) + ((y1 + 1) * SCALE)
	draw.Draw(img, image.Rect(x0, y0, x1, y1), color, image.ZP, mode)
}

func main() {
	m, err := NewMap(os.Args[1])
	if err != nil {
		panic(err)
	}

	i, err := m.Draw()
	if err != nil {
		panic(err)
	}

	o, err := os.Create("output.png")
	if err != nil {
		panic(err)
	}
	if err := png.Encode(o, i); err != nil {
		panic(err)
	}
	o.Close()
}
