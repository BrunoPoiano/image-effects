package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"strconv"
	"strings"
	"syscall/js"
	"time"

	"github.com/anthonynsimon/bild/blur"
	"github.com/anthonynsimon/bild/effect"
	"github.com/anthonynsimon/bild/transform"
)

type model struct {
	imageWidth      string
	effectSelected  string
	effectRange     string
	checkAscii      bool
	checkColor      bool
	imageSelected   js.Value
	global          js.Value
	document        js.Value
	effectsRateMap  map[string]bool
	asciiChars      string
	execChangeImage func()
}

func main() {

	g := js.Global()
	m := &model{
		asciiChars:     "░▒▓█",
		imageWidth:     "100",
		effectRange:    "3",
		effectSelected: "",
		checkAscii:     false,
		effectsRateMap: effectsRateMapFunc(),
		global:         g,
		document:       g.Get("document"),
		checkColor:     true,
	}

	//Adding debounce to changeImage func
	m.execChangeImage = debounce(200*time.Millisecond, func() {
		m.changeImage()
	})

	//getting elements
	inputZoomRange := m.document.Call("getElementById", "input-zoom-range")
	selectEffect := m.document.Call("getElementById", "select-effect")
	selectAscii := m.document.Call("getElementById", "select-ascii")
	inputEffectRange := m.document.Call("getElementById", "input-effect-range")
	inputCheckboxAscii := m.document.Call("getElementById", "input-checkbox-ascii")
	inputCheckboxColor := m.document.Call("getElementById", "input-checkbox-color")
	inputFile := m.document.Call("getElementById", "input-file")
	inputTextAscii := m.document.Call("getElementById", "input-text-ascii")

	//adding reactivity
	inputZoomRange.Call("addEventListener", "input", js.FuncOf(m.inputZoomRangeChange))
	selectEffect.Call("addEventListener", "input", js.FuncOf(m.effectChange))
	selectAscii.Call("addEventListener", "input", js.FuncOf(m.selectAsciiChange))
	inputEffectRange.Call("addEventListener", "input", js.FuncOf(m.inputEffectRangeChange))
	inputCheckboxAscii.Call("addEventListener", "input", js.FuncOf(m.inputAsciiCheckboxChange))
	inputCheckboxColor.Call("addEventListener", "input", js.FuncOf(m.inputAsciiCheckboxColor))
	inputFile.Call("addEventListener", "input", js.FuncOf(m.fileChange))
	inputTextAscii.Call("addEventListener", "input", js.FuncOf(m.inputTextAsciiChange))

	//setting default value
	inputZoomRange.Set("value", m.imageWidth)
	selectEffect.Set("value", m.effectSelected)
	selectAscii.Set("value", m.asciiChars)
	inputEffectRange.Set("value", m.effectRange)
	inputCheckboxAscii.Set("checked", m.checkAscii)
	inputCheckboxColor.Set("checked", m.checkColor)
	inputTextAscii.Set("value", m.asciiChars)

	select {}
}

func (m *model) inputAsciiCheckboxColor(this js.Value, args []js.Value) interface{} {
	m.checkColor = this.Get("checked").Bool()
	m.execChangeImage()
	return nil
}

func (m *model) inputAsciiCheckboxChange(this js.Value, args []js.Value) interface{} {
	m.checkAscii = this.Get("checked").Bool()

	inputZoomRangeDiv := m.document.Call("getElementById", "ascii-div")
	imageDiv := m.document.Call("getElementById", "img")
	asciiDiv := m.document.Call("getElementById", "ascii-art")

	changeAttribute(inputZoomRangeDiv, "data-visible", strconv.FormatBool(m.checkAscii))
	changeAttribute(imageDiv, "data-visible", strconv.FormatBool(!m.checkAscii))
	changeAttribute(asciiDiv, "data-visible", strconv.FormatBool(m.checkAscii))

	m.execChangeImage()
	return nil
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

func (m *model) inputTextAsciiChange(this js.Value, args []js.Value) interface{} {
	value := args[0].Get("target").Get("value").String()

	if value == "" || value == " " {
		value = "@%#*+=-:. "
	}

	m.asciiChars = value

	m.execChangeImage()
	return nil
}

func (m *model) selectAsciiChange(this js.Value, args []js.Value) interface{} {
	m.asciiChars = args[0].Get("target").Get("value").String()

	m.document.Call("getElementById", "input-text-ascii").
		Set("value", m.asciiChars)

	m.changeImage()
	return nil
}

func (m *model) effectChange(this js.Value, args []js.Value) interface{} {
	m.effectSelected = args[0].Get("target").Get("value").String()

	inputRateRangeDiv := m.document.Call("getElementById", "input-rate-range-div")
	dataVisible := "false"

	if m.effectsRateMap[m.effectSelected] {
		dataVisible = "true"
	}

	changeAttribute(inputRateRangeDiv, "data-visible", dataVisible)
	m.changeImage()
	return nil
}

func (m *model) inputEffectRangeChange(this js.Value, args []js.Value) interface{} {
	m.effectRange = args[0].Get("target").Get("value").String()
	m.execChangeImage()
	return nil
}

func (m *model) inputZoomRangeChange(this js.Value, args []js.Value) interface{} {
	m.imageWidth = args[0].Get("target").Get("value").String()
	m.execChangeImage()
	return nil
}

func (m *model) changeImage() {
	if m.imageSelected.IsUndefined() || m.imageSelected.IsNull() {
		return
	}

	fileReader := m.global.Get("FileReader").New()

	var onLoad js.Func
	onLoad = js.FuncOf(func(this js.Value, args []js.Value) interface{} {

		arrayBuffer := this.Get("result")

		uint8Array := m.global.Get("Uint8Array").New(arrayBuffer)

		input := make([]byte, uint8Array.Length())

		js.CopyBytesToGo(input, uint8Array)

		var img image.Image = nil
		var err error = nil

		if m.imageSelected.Get("type").String() == "image/jpeg" {
			img, err = jpeg.Decode(bytes.NewReader(input))
		} else {
			img, _, err = image.Decode(bytes.NewReader(input))
		}
		if err != nil {
			m.global.Call("alert", "Image not supportaded")
			return nil
		}

		value, _ := strconv.ParseFloat(m.effectRange, 64)
		imgWithEffects := applyEffects(img, m.effectSelected, value)

		if m.checkAscii {
			value, _ := strconv.Atoi(m.imageWidth)
			m.asciiGenerator(imgWithEffects, value)
		} else {
			m.imageEffectGenerator(imgWithEffects)
		}

		onLoad.Release()
		return nil
	})

	fileReader.Set("onload", onLoad)
	fileReader.Call("readAsArrayBuffer", m.imageSelected)
}

func (m *model) imageEffectGenerator(img image.Image) {

	var buf bytes.Buffer
	png.Encode(&buf, img)

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

func (m *model) asciiGenerator(img image.Image, width int) {
	density := []rune(m.asciiChars)

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

			if m.checkColor {
				r, g, b, _ := px.RGBA()
				colorCSS := fmt.Sprintf("rgb(%d,%d,%d)", r>>8, g>>8, b>>8)
				builder.WriteString(fmt.Sprintf(`<i style="color:%s">%c</i>`, colorCSS, density[int(charIndex)]))
			} else {
				builder.WriteRune([]rune(string(density[int(charIndex)]))[0])
			}

		}
		builder.WriteRune('\n')
	}

	asciiDiv := m.document.Call("getElementById", "ascii-art")
	asciiDiv.Set("innerHTML", builder.String())
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

func changeAttribute(content js.Value, attribute string, value string) {
	content.Call("setAttribute", attribute, value)
}

func effectsRateMapFunc() map[string]bool {
	effects := []string{"gaussianBlur", "blur", "Dilate", "edgeDetection", "erode", "median"}
	effectsMap := make(map[string]bool)
	for _, effect := range effects {
		effectsMap[effect] = true
	}

	return effectsMap
}

func debounce(duration time.Duration, fn func()) func() {
	var timer *time.Timer

	return func() {
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(duration, fn)
	}
}
