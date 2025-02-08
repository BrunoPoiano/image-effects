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

	"github.com/anthonynsimon/bild/adjust"
	"github.com/anthonynsimon/bild/blend"
	"github.com/anthonynsimon/bild/blur"
	"github.com/anthonynsimon/bild/effect"
	"github.com/anthonynsimon/bild/noise"
	"github.com/anthonynsimon/bild/segment"
	"github.com/anthonynsimon/bild/transform"
)

type model struct {
	imageWidth      int
	effectSelected  string
	effectRange     float64
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
		imageWidth:     100,
		effectRange:    0,
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
	inputCheckboxAscii := m.document.Call("getElementById", "input-checkbox-ascii")
	inputCheckboxColor := m.document.Call("getElementById", "input-checkbox-color")
	inputEffectRange := m.document.Call("getElementById", "input-effect-range")
	inputTextAscii := m.document.Call("getElementById", "input-text-ascii")
	inputZoomRange := m.document.Call("getElementById", "input-zoom-range")
	selectEffect := m.document.Call("getElementById", "select-effect")
	selectAscii := m.document.Call("getElementById", "select-ascii")
	inputFile := m.document.Call("getElementById", "input-file")

	//adding reactivity
	inputCheckboxAscii.Call("addEventListener", "input", js.FuncOf(m.inputAsciiCheckboxChange))
	inputCheckboxColor.Call("addEventListener", "input", js.FuncOf(m.inputAsciiCheckboxColor))
	inputEffectRange.Call("addEventListener", "input", js.FuncOf(m.inputEffectRangeChange))
	inputTextAscii.Call("addEventListener", "input", js.FuncOf(m.inputTextAsciiChange))
	inputZoomRange.Call("addEventListener", "input", js.FuncOf(m.inputZoomRangeChange))
	selectEffect.Call("addEventListener", "input", js.FuncOf(m.effectChange))
	selectAscii.Call("addEventListener", "input", js.FuncOf(m.selectAsciiChange))
	inputFile.Call("addEventListener", "input", js.FuncOf(m.fileChange))

	//setting default value
	inputCheckboxAscii.Set("checked", m.checkAscii)
	inputCheckboxColor.Set("checked", m.checkColor)
	inputEffectRange.Set("value", m.effectRange)
	inputTextAscii.Set("value", m.asciiChars)
	inputZoomRange.Set("value", m.imageWidth)
	selectEffect.Set("value", m.effectSelected)
	selectAscii.Set("value", m.asciiChars)

	select {}
}

func (m *model) inputAsciiCheckboxColor(this js.Value, args []js.Value) interface{} {
	m.checkColor = this.Get("checked").Bool()
	m.execChangeImage()
	return nil
}

func (m *model) inputAsciiCheckboxChange(this js.Value, args []js.Value) interface{} {
	m.checkAscii = this.Get("checked").Bool()
	m.execChangeImage()
	return nil
}

func (m *model) fileChange(this js.Value, args []js.Value) interface{} {
	files := args[0].Get("target").Get("files")

	if files.Length() > 0 {
		m.imageSelected = files.Index(0)
		m.changeImage()
	}
	return nil
}

func (m *model) inputTextAsciiChange(this js.Value, args []js.Value) interface{} {
	m.asciiChars = args[0].Get("target").Get("value").String()
	m.execChangeImage()
	return nil
}

func (m *model) selectAsciiChange(this js.Value, args []js.Value) interface{} {
	m.asciiChars = args[0].Get("target").Get("value").String()
	m.document.Call("getElementById", "input-text-ascii").Set("value", m.asciiChars)
	m.changeImage()
	return nil
}

func (m *model) effectChange(this js.Value, args []js.Value) interface{} {
	m.effectSelected = args[0].Get("target").Get("value").String()
	m.effectRange = 0

	if m.effectsRateMap[m.effectSelected] {
		m.updateEffectRange("0", "10", "1")
	} else {
		switch m.effectSelected {
		case "brightness", "contrast", "saturation":
			m.updateEffectRange("-1", "1", "0.1")
		case "hue":
			m.updateEffectRange("-360", "360", "1")
		case "gamma":
			m.updateEffectRange("1", "5", "0.2")
		case "threshold":
			m.updateEffectRange("0", "200", "1")
		case "noisePerlin":
			m.updateEffectRange("0", "1", "0.01")
		case "shearH", "shearV":
			m.updateEffectRange("0", "180", "1")
		default:
			inputRateRangeDiv := m.document.Call("getElementById", "input-rate-range-div")
			changeAttribute(inputRateRangeDiv, "data-visible", "false")
		}
	}

	m.changeImage()
	return nil
}

func (m model) updateEffectRange(min string, max string, step string) {
	inputRange := m.document.Call("getElementById", "input-effect-range")
	inputRange.Set("value", "0")
	inputRange.Set("min", min)
	inputRange.Set("max", max)
	inputRange.Set("step", step)

	inputRateRangeDiv := m.document.Call("getElementById", "input-rate-range-div")
	changeAttribute(inputRateRangeDiv, "data-min", min)
	changeAttribute(inputRateRangeDiv, "data-max", max)
	changeAttribute(inputRateRangeDiv, "data-visible", "true")

}

func (m *model) inputEffectRangeChange(this js.Value, args []js.Value) interface{} {
	value := args[0].Get("target").Get("value").String()
	m.effectRange, _ = strconv.ParseFloat(value, 64)
	m.execChangeImage()
	return nil
}

func (m *model) inputZoomRangeChange(this js.Value, args []js.Value) interface{} {
	value := args[0].Get("target").Get("value").String()
	m.imageWidth, _ = strconv.Atoi(value)
	m.execChangeImage()
	return nil
}

func (m *model) changeImage() {
	if m.imageSelected.IsUndefined() || m.imageSelected.IsNull() {
		return
	}

	if m.asciiChars == "" || m.asciiChars == " " {
		return
	}

	fileReader := m.global.Get("FileReader").New()

	var onLoad js.Func
	onLoad = js.FuncOf(func(this js.Value, args []js.Value) interface{} {

		uint8Array := m.global.Get("Uint8Array").New(this.Get("result"))
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

		imgWithEffects := m.applyEffects(img)

		if m.checkAscii {
			m.asciiGenerator(imgWithEffects)
		} else {
			m.imageEffectGenerator(imgWithEffects)
		}

		onLoad.Release()

		inputZoomRangeDiv := m.document.Call("getElementById", "ascii-div")
		imageDiv := m.document.Call("getElementById", "img")
		asciiDiv := m.document.Call("getElementById", "ascii-art")

		changeAttribute(inputZoomRangeDiv, "data-visible", strconv.FormatBool(m.checkAscii))
		changeAttribute(imageDiv, "data-visible", strconv.FormatBool(!m.checkAscii))
		changeAttribute(asciiDiv, "data-visible", strconv.FormatBool(m.checkAscii))

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

func (m *model) asciiGenerator(img image.Image) {
	density := []rune(m.asciiChars)

	width, height := resizeImg(img, m.imageWidth)
	resul := transform.Resize(img, width, height, transform.Linear)
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

func (m model) applyEffects(img image.Image) image.Image {
	var result image.Image = img

	switch m.effectSelected {
	case "gaussianBlur":
		result = blur.Gaussian(result, float64(m.effectRange))
	case "blur":
		result = blur.Box(result, float64(m.effectRange))
	case "Dilate":
		result = effect.Dilate(result, float64(m.effectRange))
	case "edgeDetection":
		result = effect.EdgeDetection(result, float64(m.effectRange))
	case "emboss":
		result = effect.Emboss(result)
	case "erode":
		result = effect.Erode(result, float64(m.effectRange))
	case "grayscale":
		result = effect.Grayscale(result)
	case "invert":
		result = effect.Invert(result)
	case "median":
		result = effect.Median(result, float64(m.effectRange))
	case "sepia":
		result = effect.Sepia(result)
	case "sharpen":
		result = effect.Sharpen(result)
	case "sobale":
		result = effect.Sobel(result)

	case "brightness":
		result = adjust.Brightness(result, float64(m.effectRange))
	case "contrast":
		result = adjust.Contrast(result, float64(m.effectRange))
	case "gamma":
		result = adjust.Gamma(result, float64(m.effectRange))
	case "hue":
		result = adjust.Hue(result, int(m.effectRange))
	case "saturation":
		result = adjust.Saturation(result, float64(m.effectRange))
	case "threshold":
		result = segment.Threshold(result, uint8(m.effectRange))
	case "flipH":
		result = transform.FlipH(result)
	case "flipV":
		result = transform.FlipV(result)
	case "shearH":
		result = transform.ShearH(result, float64(m.effectRange))
	case "shearV":
		result = transform.ShearV(result, float64(m.effectRange))

	case "noiseUniformColored":
		imgBounds := result.Bounds()
		noise := noise.Generate(imgBounds.Dx(), imgBounds.Dy(), &noise.Options{Monochrome: false, NoiseFn: noise.Uniform})
		result = blend.Overlay(noise, result)
	case "noiseBinaryMonochrome":
		imgBounds := result.Bounds()
		noise := noise.Generate(imgBounds.Dx(), imgBounds.Dy(), &noise.Options{Monochrome: true, NoiseFn: noise.Binary})
		result = blend.Opacity(noise, result, 0.5)
	case "noiseGaussianMonochrome":
		imgBounds := result.Bounds()
		noise := noise.Generate(imgBounds.Dx(), imgBounds.Dy(), &noise.Options{Monochrome: true, NoiseFn: noise.Gaussian})
		result = blend.Overlay(noise, result)
	case "noisePerlin":
		imgBounds := result.Bounds()
		noise := noise.GeneratePerlin(imgBounds.Dx(), imgBounds.Dy(), m.effectRange)
		result = blend.Overlay(noise, result)
	}

	return result
}

func resizeImg(img image.Image, newWidth int) (int, int) {
	imgBounds := img.Bounds()
	aspectRatio := float64(newWidth) / float64(imgBounds.Dx())
	newHeight := int(float64(imgBounds.Dy()) * aspectRatio)

	return newWidth, newHeight
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
