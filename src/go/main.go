package main

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/anthonynsimon/bild/blur"
	"github.com/anthonynsimon/bild/effect"
	"github.com/anthonynsimon/bild/transform"
)

type model struct {
	imageWidth     string
	effectSelected string
	effectRange    string
	imageSelected  js.Value
	global         js.Value
	document       js.Value
}

func main() {
	println("ContentLoaded")
	g := js.Global()
	m := &model{
		imageWidth:     "100",
		effectRange:    "3",
		effectSelected: "ascii",
		global:         g,
		document:       g.Get("document"),
	}

	m.document.Call("getElementById", "input-zoom-range").
		Call("addEventListener", "input", js.FuncOf(m.inputZoomRangeChange))

	m.document.Call("getElementById", "select-effect").
		Call("addEventListener", "input", js.FuncOf(m.effectChange))

	m.document.Call("getElementById", "input-effect-range").
		Call("addEventListener", "input", js.FuncOf(m.inputEffectRangeChange))

	m.document.Call("getElementById", "input-file").
		Call("addEventListener", "input", js.FuncOf(m.fileChange))

	select {}
}

func (m *model) fileChange(this js.Value, args []js.Value) interface{} {
	files := args[0].Get("target").Get("files")

	if files.Length() > 0 {
		file := files.Index(0)
		m.imageSelected = file
		m.changeImage()
	}
	return nil
}

func (m *model) effectChange(this js.Value, args []js.Value) interface{} {
	m.effectSelected = args[0].Get("target").Get("value").String()
	inputZoomRangeDiv := m.document.Call("getElementById", "input-zoom-range-div")
	dataVisible := "false"

	if m.effectSelected == "ascii" {
		dataVisible = "true"
	}
	changeAttribute(inputZoomRangeDiv, "data-visible", dataVisible)

	m.changeImage()
	return nil
}

func (m *model) inputEffectRangeChange(this js.Value, args []js.Value) interface{} {
	m.effectRange = args[0].Get("target").Get("value").String()
	println(m.effectRange)
	m.changeImage()
	return nil
}

func (m *model) inputZoomRangeChange(this js.Value, args []js.Value) interface{} {
	m.imageWidth = args[0].Get("target").Get("value").String()
	m.changeImage()
	return nil
}

func (m *model) changeImage() {
	println("changeImage")
	if m.imageSelected.IsUndefined() || m.imageSelected.IsNull() {
		println("imageNull")
		return
	}

	contentDiv := m.document.Call("getElementById", "content-div")

	changeAttribute(contentDiv, "data-loading", "true")
	//clear ascii pre
	m.document.Call("getElementById", "ascii-art").
		Set("innerHTML", "")
	m.document.Call("getElementById", "img").Set("src", "")

	fileReader := m.global.Get("FileReader").New()

	println("onLoad js.Func")
	var onLoad js.Func
	onLoad = js.FuncOf(func(this js.Value, args []js.Value) interface{} {

		println("onLoad")
		arrayBuffer := this.Get("result")

		uint8Array := m.global.Get("Uint8Array").New(arrayBuffer)

		input := make([]byte, uint8Array.Length())

		println("CopyBytesToGo")
		js.CopyBytesToGo(input, uint8Array)

		var img image.Image = nil
		var err error = nil

		if m.imageSelected.Get("type").String() == "image/jpeg" {
			img, err = jpeg.Decode(bytes.NewReader(input))
		} else {
			img, _, err = image.Decode(bytes.NewReader(input))
		}
		if err != nil {
			m.global.Call("alert", "Error decoding image")
			return nil
		}

		switch m.effectSelected {

		case "ascii":
			value, _ := strconv.Atoi(m.imageWidth)

			println("asciiGenerator")
			m.asciiGenerator(img, value)
		default:

			println("imageEffectGenerator")
			m.imageEffectGenerator(img)
		}
		onLoad.Release()
		changeAttribute(contentDiv, "data-loading", "false")
		return nil
	})

	fileReader.Set("onload", onLoad)
	fileReader.Call("readAsArrayBuffer", m.imageSelected)
}

func (m *model) imageEffectGenerator(img image.Image) {
	value, _ := strconv.ParseFloat(m.effectRange, 64)
	println(value)
	result := applyEffects(img, m.effectSelected, value)

	var buf bytes.Buffer
	png.Encode(&buf, result)

	data := buf.Bytes()

	uint8Array := m.global.Get("Uint8Array").New(len(data))

	js.CopyBytesToJS(uint8Array, data)

	array := m.global.Get("Array").New(1)
	array.SetIndex(0, uint8Array)

	blobOpt := m.global.Get("Object").New()
	blobOpt.Set("type", "image/png")
	blob := m.global.Get("Blob").New(array, blobOpt)

	url := m.global.Get("URL").Call("createObjectURL", blob)
	m.document.Call("getElementById", "img").Set("src", url)

}

func applyEffects(img image.Image, effectString string, rate float64) image.Image {
	var result image.Image = img

	switch effectString {
	case "gaussianBlur":
		result = blur.Gaussian(result, rate)
	case "blur":
		result = blur.Box(result, rate)
	case "Dilate":
		result = effect.Dilate(result, rate)
	case "edgeDetection":
		result = effect.EdgeDetection(result, rate)
	case "emboss":
		result = effect.Emboss(result)
	case "erode":
		result = effect.Erode(result, rate)
	case "grayscale":
		result = effect.Grayscale(img)
	case "invert":
		result = effect.Invert(img)
	case "median":
		result = effect.Median(img, rate)
	case "sepia":
		result = effect.Sepia(img)
	case "sharpen":
		result = effect.Sharpen(img)
	case "sobale":
		result = effect.Sobel(img)
	}

	return result

}

func resizeImg(img image.Image, newWidth int) image.Image {

	imgBounds := img.Bounds()

	aspectRatio := float64(newWidth) / float64(imgBounds.Dx())
	newHeight := int(float64(imgBounds.Dy()) * aspectRatio)

	return transform.Resize(img, newWidth, newHeight, transform.Linear)
}

func (m *model) asciiGenerator(img image.Image, width int) {

	density := []rune("Ñ@#W$9876543210?!abc;:+=-,._ ")

	resul := resizeImg(img, width)
	bounds := resul.Bounds()
	var builder strings.Builder

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			px := resul.At(x, y)
			gr := color.GrayModel.Convert(px)
			gray := gr.(color.Gray)

			intensity := float64(gray.Y) / 255.0
			charIndex := math.Floor(float64(len(density)-1) * intensity)
			char := density[int(charIndex)]
			builder.WriteRune([]rune(string(char))[0])
		}
		builder.WriteRune('\n')
	}

	asciiDiv := m.document.Call("getElementById", "ascii-art")
	asciiDiv.Set("innerHTML", builder.String())
}

func changeAttribute(content js.Value, attribute string, value string) {
	content.Call("setAttribute", attribute, value)
}
