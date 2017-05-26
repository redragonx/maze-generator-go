package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"image"
	"math/big"
	"os"
	"time"

	_ "image/png"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"
	"github.com/pkg/profile"

	"golang.org/x/image/colornames"
	"src.rocks/redragonx/maze-generator-go/stack"
)

var visitedColor = pixel.RGB(0.5, 0, 1).Mul(pixel.Alpha(0.35))
var hightlightColor = pixel.RGB(0.3, 0, 0).Mul(pixel.Alpha(0.45))

func loadPicture(path string) (pixel.Picture, error) {
	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	defer file.Close()
	img, _, err := image.Decode(file)

	if err != nil {
		return nil, err
	}

	return pixel.PictureDataFromImage(img), nil
}

type Cell struct {
	// Wall order
	// top, right, bottom, left
	walls [4]bool

	row     int
	col     int
	visited bool
}

func (c *Cell) Draw(imd *imdraw.IMDraw, wallSize int) {
	drawCol := c.col * wallSize // x
	drawRow := c.row * wallSize // y

	// fmt.Printf("row: %d, col: %d", c.row, c.col)
	imd.Color = colornames.White
	if c.walls[0] {
		// top line
		imd.Push(pixel.V(float64(drawCol), float64(drawRow)), pixel.V(float64(drawCol+wallSize), float64(drawRow)))
		imd.Line(3)
	}
	if c.walls[1] {
		// right Line
		// imd.Color(colornames.Red)
		imd.Push(pixel.V(float64(drawCol+wallSize), float64(drawRow)), pixel.V(float64(drawCol+wallSize), float64(drawRow+wallSize)))
		imd.Line(3)
	}
	if c.walls[2] {

		//  bottom line
		//imd.Color(colornames.Beige)
		imd.Push(pixel.V(float64(drawCol+wallSize), float64(drawRow+wallSize)), pixel.V(float64(drawCol), float64(drawRow+wallSize)))
		imd.Line(3)
	}
	if c.walls[3] {
		// left line
		imd.Push(pixel.V(float64(drawCol), float64(drawRow+wallSize)), pixel.V(float64(drawCol), float64(drawRow)))
		imd.Line(3)
	}
	imd.EndShape = imdraw.SharpEndShape

	if c.visited {

		imd.Color = visitedColor
		imd.Push(pixel.V(float64(drawCol), (float64(drawRow))), pixel.V(float64(drawCol+wallSize), float64(drawRow+wallSize)))
		imd.Rectangle(0)
	}
}

func index(i, j, cols, rows int) int {
	if i < 0 || j < 0 || i > cols-1 || j > rows-1 {
		return -1
	}
	return i + j*cols
}

func getCellAt(i int, j int, cols int, rows int, grid []*Cell) (*Cell, error) {
	possibleIndex := index(i, j, cols, rows)

	if possibleIndex == -1 {
		return nil, fmt.Errorf("index: index is a negative number %d", possibleIndex)
	}

	return grid[possibleIndex], nil
}

func (c *Cell) GetNeighbors(grid []*Cell, cols int, rows int) ([]*Cell, error) {
	neighbors := []*Cell{}
	j := c.row
	i := c.col

	top, _ := getCellAt(i, j-1, cols, rows, grid)
	right, _ := getCellAt(i+1, j, cols, rows, grid)
	bottom, _ := getCellAt(i, j+1, cols, rows, grid)
	left, _ := getCellAt(i-1, j, cols, rows, grid)

	if top != nil && !top.visited {
		neighbors = append(neighbors, top)
	}
	if right != nil && !right.visited {
		neighbors = append(neighbors, right)
	}
	if bottom != nil && !bottom.visited {
		neighbors = append(neighbors, bottom)
	}
	if left != nil && !left.visited {
		neighbors = append(neighbors, left)
	}

	if len(neighbors) > 0 {
		return neighbors, nil
	} else {
		return nil, errors.New("We checked all cells...")
	}
}

func (c *Cell) GetRandomNeighbor(grid []*Cell, cols int, rows int) (*Cell, error) {
	neighbors, err := c.GetNeighbors(grid, cols, rows)
	if neighbors != nil {
		nBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(neighbors))))
		if err != nil {
			panic(err)
		}
		randomIndex := nBig.Int64()
		// fmt.Printf("random index: %d", randomIndex)
		return neighbors[randomIndex], nil
	} else {
		return nil, err
	}
}

func (c *Cell) hightlight(imd *imdraw.IMDraw, wallSize int) {
	x := c.col * wallSize
	y := c.row * wallSize

	imd.Color = hightlightColor
	imd.Push(pixel.V(float64(x), float64(y)), pixel.V(float64(x+wallSize), float64(y+wallSize)))
	imd.Rectangle(0)
}

func NewCell(col int, row int) *Cell {
	newCell := new(Cell)
	newCell.row = row
	newCell.col = col

	for i := range newCell.walls {
		newCell.walls[i] = true
	}

	return newCell

}

func initGrid(grid []*Cell, cols int, rows int) []*Cell {
	for j := 0; j < rows; j++ {
		for i := 0; i < cols; i++ {
			newCell := NewCell(i, j)
			grid = append(grid, newCell)
		}
	}
	// fmt.Printf("%d", len(grid))
	return grid

}

func run() {

	var (
		// In pixels
		// 10x10 wallgrid
		wallSize    = 40
		screenHeigt = 800
		screenWidth = 800

		frames = 0
		second = time.Tick(time.Second)

		grid           = []*Cell{}
		cols           = screenWidth / wallSize
		rows           = screenHeigt / wallSize
		currentCell    = new(Cell)
		backTrackStack = stack.NewStack(50)
	)

	// Set game FPS manually
	fps := time.Tick(time.Second / 60)

	cfg := pixelgl.WindowConfig{
		Title:  "Pixel Rocks!",
		Bounds: pixel.R(0, 0, float64(screenHeigt), float64(screenWidth)),
	}

	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	// Make an empty grid
	grid = initGrid(grid, cols, rows)
	currentCell = grid[0]

	//fmt.Println("Number of cells", string(len(grid)))
	gridIMDraw := imdraw.New(nil)

	for !win.Closed() {

		win.Clear(colornames.Gray)
		gridIMDraw.Clear()

		for i := range grid {
			grid[i].Draw(gridIMDraw, wallSize)
			//fmt.Printf("index: %d", i)
		}

		// step 1
		// Make the initial cell the current cell and mark it as visited
		currentCell.visited = true
		currentCell.hightlight(gridIMDraw, wallSize)

		// step 2.1
		// If the current cell has any neighbours which have not been visited
		// Choose a random unvisited cell
		nextCell, _ := currentCell.GetRandomNeighbor(grid, cols, rows)
		if nextCell != nil && !nextCell.visited {

			// step 2.2
			// Push the current cell to the stack
			backTrackStack.Push(currentCell)

			// step 2.3
			// Remove the wall between the current cell and the chosen cell

			removeWalls(currentCell, nextCell)

			// step 2.4
			// Make the chosen cell the current cell and mark it as visited
			nextCell.visited = true
			currentCell = nextCell
		} else if backTrackStack.Len() > 0 {
			currentCell = backTrackStack.Pop().(*Cell)
		}

		gridIMDraw.Draw(win)
		win.Update()
		<-fps
		updateFPSDisplay(win, &cfg, &frames, second)
	}
}

func removeWalls(a *Cell, b *Cell) {
	x := a.col - b.col

	if x == 1 {
		a.walls[3] = false
		b.walls[1] = false
	} else if x == -1 {
		a.walls[1] = false
		b.walls[3] = false
	}

	y := a.row - b.row

	if y == 1 {
		a.walls[0] = false
		b.walls[2] = false
	} else if y == -1 {
		a.walls[2] = false
		b.walls[0] = false
	}
}

func updateFPSDisplay(win *pixelgl.Window, cfg *pixelgl.WindowConfig, frames *int, second <-chan time.Time) {

	*frames++
	select {
	case <-second:
		win.SetTitle(fmt.Sprintf("%s | FPS: %d", cfg.Title, *frames))
		*frames = 0
	default:
	}

}
func main() {

	defer profile.Start().Stop()
	pixelgl.Run(run)
}
