package main

func main() {
	go checkmain()
	go networkmain()
	httpmain()
}
