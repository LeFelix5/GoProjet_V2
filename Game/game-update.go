/*
//  Implementation of the Update method for the Game structure
//  This method is called once at every frame (60 frames per second)
//  by ebiten, juste before calling the Draw method (game-draw.go).
//  Provided with a few utilitary methods:
//    - CheckArrival
//    - ChooseRunners
//    - HandleLaunchRun
//    - HandleResults
//    - HandleWelcomeScreen
//    - Reset
//    - UpdateAnimation
//    - UpdateRunners
*/

package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var serverChannel = make(chan string)
var allConnected = false
var startGame = false

func listenToServer(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		// Lire un message du serveur
		message := scanner.Text()

		// Envoyer le message reçu sur le canal
		serverChannel <- message
	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
	}
}

// HandleWelcomeScreen waits for the player to push SPACE in order to
// start the game
func (g *Game) HandleWelcomeScreen() bool {
	select {
	case message := <-serverChannel:
		fmt.Println(message)
		if strings.Contains(message, "NOMBRE_JOUEURS") {
			nbJoueurs := strings.Split(message, "|")
			allConnected = false
			g.joueurConnected = nbJoueurs[1]
		}
		if message == "Tous les joueurs sont connectés" {
			allConnected = true
		}
	default:
		// Do nothing
	}

	if allConnected {
		return inpututil.IsKeyJustPressed(ebiten.KeySpace)
	}
	return false
}

// ChooseRunners loops over all the runners to check which sprite each
// of them selected
func (g *Game) ChooseRunners() (done bool) {
	done = true
	for i := range g.runners {
		if i == 0 {
			done = g.runners[i].ManualChoose() && done
		} else {
			done = g.runners[i].RandomChoose() && done
		}
	}
	return done
}

// HandleLaunchRun countdowns to the start of a run
func (g *Game) HandleLaunchRun() bool {
	select {
	case message := <-serverChannel:
		fmt.Println(message)
		if message == "Tous les joueurs ont sélectionnés leur personnage" {
			startGame = true
			g.joueursPret = true
		} else if strings.Contains(message, "Course") {
			courseState := strings.Split(message, "|")
			if courseState[1] == "Start" {
				startGame = true
			}
		} else if strings.Contains(message, "NOMBRE_JOUEURS_WAITING") {
			nbJoueurs := strings.Split(message, "|")
			g.joueurWaiting = nbJoueurs[1]
			fmt.Println(g.joueurWaiting)
		}

	default:
		// Do nothing
	}

	if startGame {
		if time.Since(g.f.chrono).Milliseconds() > 1000 {
			g.launchStep++
			g.f.chrono = time.Now()
		}
		if g.launchStep >= 5 {
			g.launchStep = 0
			return true
		}
	}
	
	return false
}

// UpdateRunners loops over all the runners to update each of them
func (g *Game) UpdateRunners() {
	for i := range g.runners {
		if i == 0 {
			g.runners[i].ManualUpdate()
		} else {
			g.runners[i].RandomUpdate()
		}
	}
}

// CheckArrival loops over all the runners to check which ones are arrived
func (g *Game) CheckArrival() (finished bool) {
	finished = true
	for i := range g.runners {
		g.runners[i].CheckArrival(&g.f)
		finished = finished && g.runners[i].arrived
	}
	return finished
}

// Reset resets all the runners and the field in order to start a new run
func (g *Game) Reset() {
	for i := range g.runners {
		g.runners[i].Reset(&g.f)
	}
	g.f.Reset()
}

// UpdateAnimation loops over all the runners to update their sprite
func (g *Game) UpdateAnimation() {
	for i := range g.runners {
		g.runners[i].UpdateAnimation(g.runnerImage)
	}
}

// HandleResults computes the resuls of a run and prepare them for
// being displayed
func (g *Game) HandleResults() bool {
	select {
	case message := <-serverChannel:
		if strings.Contains(message, "Time") {
			startGame = false
			time := strings.Split(message, "|")
			g.ranking = strings.Split(time[1], ",")
			fmt.Println("Score -",time[1])
		}
	default:
		// Do nothing
	}

	if !startGame {
		if time.Since(g.f.chrono).Milliseconds() > 1000 || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.resultStep++
			g.f.chrono = time.Now()
		}
		if g.resultStep >= 4 && inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.resultStep = 0
			return true
		}	
	}
	
	return false
}

// Update is the main update function of the game. It is called by ebiten
// at each frame (60 times per second) just before calling Draw (game-draw.go)
// Depending of the current state of the game it calls the above utilitary
// function and then it may update the state of the game
func (g *Game) Update() error {
	switch g.state {
	case StateWelcomeScreen:
		done := g.HandleWelcomeScreen()
		if done {
			g.state++
		}
	case StateChooseRunner:
		done := g.ChooseRunners()
		if done {
			// Envoie un message au serveur
			message := "Personnage sélectionné!"
			_, err := g.conn.Write([]byte(message))
			if err != nil {
				fmt.Println("Erreur lors de l'envoi du message:", err)
			}

			g.UpdateAnimation()
			g.state++
		}
	case StateLaunchRun:
		done := g.HandleLaunchRun()
		
		if !g.joueursPret && inpututil.IsKeyJustPressed(ebiten.KeyUp){
			message := "Personnage désélectionné!"
			_, err := g.conn.Write([]byte(message))
				if err != nil {
					fmt.Println("Erreur lors de l'envoi du message:", err)
				}
			g.runners[0].colorSelected = false
			g.state--
		}
		if done {
			g.state++
		}
	case StateRun:
		g.UpdateRunners()
		finished := g.CheckArrival()
		g.UpdateAnimation()
		if finished {
			s, ms := GetSeconds(g.runners[0].runTime.Milliseconds())

			// Envoie le temps d'arrivée du joueur au serveur
			message := fmt.Sprint("Time|", s, ":", ms)
			fmt.Println(message)
			_, err := g.conn.Write([]byte(message))
			if err != nil {
				fmt.Println("Erreur lors de l'envoi du message:", err)
			}

			g.state++			
		}
	case StateResult:
		done := g.HandleResults()
		if done {
			g.Reset()
			g.state = StateLaunchRun
			
			// Notifie le serveur que la course reprend
			message := fmt.Sprint("Course|Wait")
			fmt.Println(message)
			_, err := g.conn.Write([]byte(message))
			if err != nil {
				fmt.Println("Erreur lors de l'envoi du message:", err)
			}
		}
	}
	return nil
}
