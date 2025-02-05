/*
 *
document.addEventListener("DOMContentLoaded", () => {
  let image = null;
  let effect = "ascii";
  let width = "100";

  function fileChange(event) {
    const input = event.target;
    image = input.files[0];

    applyEffect();
  }

  function effectChange(event) {
    effect = event.target.value;
    applyEffect();
  }
  function rangeChange(event) {
    width = event.target.value;
    applyEffect();
  }

  function applyEffect() {
    if (image == null) return;

    const reader = new FileReader();
    reader.readAsArrayBuffer(image);
    reader.onloadend = (e) => {
      const data = new Uint8Array(reader.result);
      const imageEffect = changeImage(data, image.type, effect, width);
      const blob = new Blob([imageEffect], { type: "image/png" });
      document.getElementById("img").src = URL.createObjectURL(blob);
    };
  }

  document.getElementById("input-file").addEventListener("change", fileChange);
  document
    .getElementById("select-effect")
    .addEventListener("change", effectChange);
});


 */
