const express = require("express");
const path = require("path");
const app = express();

app.use(express.static(path.join(__dirname, "src")));

app.get("/", (request, resp) => {
	resp.sendFile(path.join(__dirname, "src", "index.html"));
});

app.listen(3000, () => {
	console.log("running 3000 port");
});
