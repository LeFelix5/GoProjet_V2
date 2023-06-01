package main

import (
	"fmt"
	"net"
	"strings"
)

func main() {
    // Crée un slice pour stocker les connexions des joueurs
    var joueurs []net.Conn
    var characterChosen int
    var runnerFinished int
    var runnerStarted int
    var joueurConnected int
    var ranking []string


    // Démarre le serveur sur le port 8080
    ln, err := net.Listen("tcp", ":8080")
    if err != nil {
        fmt.Println(err)
        return
    }
    defer ln.Close()

    // Attend les connexions des joueurs
    for {
        conn, err := ln.Accept()
        if err != nil {
            fmt.Println(err)
            continue
        }
        if len(joueurs) < 4 {
            // Ajoute la connexion du joueur à la liste
            joueurs = append(joueurs, conn)

            // Averti le joueur qu'il est connecté
            conn.Write([]byte("Vous êtes connecté\n"))
            joueurConnected++
            

            // Averti les autres joueurs qu'un nouveau joueur s'est connecté
            for _, joueur := range joueurs {
                message := fmt.Sprint("NOMBRE_JOUEURS|", joueurConnected, "\n")
                joueur.Write([]byte(message))
            }
			// Vérifi si quatre joueurs sont connectés
            if len(joueurs) == 4 {
                // Prévient les joueurs qu'ils sont tous connectés
                for _, joueur := range joueurs {
                    joueur.Write([]byte("Tous les joueurs sont connectés\n"))
                }
            }

            // Lit les messages envoyés par le joueur
            go func(conn net.Conn) {
                defer conn.Close()
                for {
                    buffer := make([]byte, 1024)
                    n, err := conn.Read(buffer) 

                    if err != nil {
                        // Averti les autres joueurs qu'un joueur s'est déconnecté
                        for i, joueur := range joueurs {
                            if joueur == conn {
                                joueurs = append(joueurs[:i], joueurs[i+1:]...)

                                if characterChosen > 0 {
                                    characterChosen--
                                }
                                if runnerFinished > 0 {
                                    runnerFinished--
                                }
                                if runnerStarted > 0 {
                                    runnerStarted--
                                } 

                                break
                            }
                        }
                        joueurConnected--
                        for _, joueur := range joueurs {
                            message := fmt.Sprint("NOMBRE_JOUEURS|", joueurConnected, "\n")
                            joueur.Write([]byte(message))
                        }
                        return
                    }

                    message := string(buffer[:n])
                    // Réceptionne le message de sélection des personnages
                    if message == "Personnage sélectionné!" {
                        characterChosen++
                        if characterChosen > 1 {
                            fmt.Println(characterChosen, "joueurs ont sélectionnés leur personnage")
                        } else {
                            fmt.Println(characterChosen, "joueur a sélectionné son personnage")
                        }

                        // Prévient les joueurs qu'ils ont tous sélectionnés leur personnage
                        if characterChosen == 4 {
                            for _, joueur := range joueurs {
                                joueur.Write([]byte("Tous les joueurs ont sélectionnés leur personnage\n"))
                            }
                        }
                    }
                    if message == "Personnage désélectionné!"{
                        characterChosen--
                    }

                    // Réceptionne le message du temps des joueurs
                    if strings.Contains(message, "Time") {
                        runnerFinished++
                        time := strings.Split(message, "|")
                        ranking = append(ranking, time[1])
                        fmt.Println("Le joueur", runnerFinished, "est arrivé en", time[1])

                        if runnerFinished == 4 {
                            for _, joueur := range joueurs {
                                message := fmt.Sprint("NOMBRE_JOUEURS|", joueurConnected, "\n")
                                joueur.Write([]byte(message))

                                // Envoi la liste des temps aux joueurs
                                messageTime := fmt.Sprint("Time|", strings.Join(ranking, ","), "\n")
                                joueur.Write([]byte(messageTime))                                           
                            }                        
                        }
                    }

                    if strings.Contains(message, "Course") {
                        courseState := strings.Split(message, "|")
                        if courseState[1] == "Wait" {
                            runnerStarted++
                            for _, joueur := range joueurs {
                                message := fmt.Sprint("NOMBRE_JOUEURS_WAITING|", runnerStarted, "\n")
                                joueur.Write([]byte(message))
                            }
                        }

                        if runnerStarted == 4 {
                            runnerFinished = 0
                            runnerStarted = 0
                            for _, joueur := range joueurs {
                                joueur.Write([]byte("Course|Start\n"))
                            }
                        }
                    }
                }
            }(conn)
        } else {
            conn.Write([]byte("Le serveur est plein!\n"))
            conn.Close()
        }
    } 
}