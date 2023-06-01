/*
// Implementation of a main function setting a few characteristics of
// the game window, creating a game, and launching it
*/

package main

import (
	"flag"
	"fmt"
	_ "image/png"
	"log"
	"net"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenWidth  = 800 // Width of the game window (in pixels)
	screenHeight = 160 // Height of the game window (in pixels)
)

func main() {
	var getTPS bool

	flag.BoolVar(&getTPS, "tps", false, "Afficher le nombre d'appel à Update par seconde")
	ip := flag.String("ip", "localhost", "Adresse IP du serveur")
	flag.Parse()

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("LP MiAR -- Programmation répartie (UE03EC2)")

	// Connexion au serveur
	conn, error := net.Dial("tcp", *ip+":8080")
	if error != nil {
		fmt.Println(error)
	}
	defer conn.Close()

	g := InitGame()
	g.getTPS = getTPS
	g.conn = conn

	go listenToServer(g.conn)

	err := ebiten.RunGame(&g)
	log.Print(err)

}
