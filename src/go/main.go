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

var imageWidth string = "100"
var effectSelected string = "ascii"
var imageSelected js.Value
var global = js.Global()
var document = global.Get("document")

func main() {
	println("ContentLoaded")

	document.Call("getElementById", "input-range").
		Call("addEventListener", "input", js.FuncOf(rangeChange))

	document.Call("getElementById", "select-effect").
		Call("addEventListener", "input", js.FuncOf(effectChange))

	document.Call("getElementById", "input-file").
		Call("addEventListener", "input", js.FuncOf(fileChange))

	select {}
}

func fileChange(this js.Value, args []js.Value) interface{} {
	files := args[0].Get("target").Get("files")

	if files.Length() > 0 {
		file := files.Index(0)
		imageSelected = file
		changeImage()
	}
	return nil
}

func effectChange(this js.Value, args []js.Value) interface{} {
	effectSelected = args[0].Get("target").Get("value").String()
	if effectSelected == "ascii" {
		document.Call("getElementById", "input-range").Call("setAttribute", "data-visible", "true")
	} else {
		document.Call("getElementById", "input-range").Call("setAttribute", "data-visible", "false")
	}
	changeImage()
	return nil
}
func rangeChange(this js.Value, args []js.Value) interface{} {
	imageWidth = args[0].Get("target").Get("value").String()
	changeImage()
	return nil
}

func changeImage() {

	if imageSelected.IsUndefined() || imageSelected.IsNull() {
		return
	}

	//clear ascii pre
	document.Call("getElementById", "ascii-art").
		Set("innerHTML", "")
	document.Call("getElementById", "img").Set("src", "")

	fileReader := global.Get("FileReader").New()

	var onLoad js.Func
	onLoad = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		arrayBuffer := this.Get("result")

		uint8Array := global.Get("Uint8Array").New(arrayBuffer)

		input := make([]byte, uint8Array.Length())

		println("CopyBytesToGo")
		js.CopyBytesToGo(input, uint8Array)

		var img image.Image = nil
		var err error = nil

		if imageSelected.Get("type").String() == "image/jpeg" {
			img, err = jpeg.Decode(bytes.NewReader(input))
		} else {
			img, _, err = image.Decode(bytes.NewReader(input))
		}
		if err != nil {
			global.Call("alert", "Error decoding image")
			return nil
		}

		switch effectSelected {

		case "ascii":
			value, _ := strconv.Atoi(imageWidth)
			asciiGenerator(img, value)
		default:
			imageEffectGenerator(img)
		}
		onLoad.Release()
		return nil
	})

	fileReader.Set("onload", onLoad)
	fileReader.Call("readAsArrayBuffer", imageSelected)
}

func imageEffectGenerator(img image.Image) {
	result := applyEffects(img, effectSelected, 3.0)

	var buf bytes.Buffer
	png.Encode(&buf, result)

	data := buf.Bytes()

	uint8Array := global.Get("Uint8Array").New(len(data))

	js.CopyBytesToJS(uint8Array, data)

	array := global.Get("Array").New(1)
	array.SetIndex(0, uint8Array)

	blobOpt := global.Get("Object").New()
	blobOpt.Set("type", "image/png")
	blob := global.Get("Blob").New(array, blobOpt)

	url := global.Get("URL").Call("createObjectURL", blob)
	document.Call("getElementById", "img").Set("src", url)

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
		result = effect.Median(img, 10.0)
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

func asciiGenerator(img image.Image, width int) {

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

	asciiDiv := document.Call("getElementById", "ascii-art")
	asciiDiv.Set("innerHTML", builder.String())
}
