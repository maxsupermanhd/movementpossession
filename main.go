package main

import (
	"errors"
	"flag"
	"fmt"
	"image/color"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gempir/go-twitch-irc/v4"
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/maxsupermanhd/lac/v2"
)

var (
	windowWidth  = int32(800)
	windowHeight = int32(150)
)

const (
	fontSize = 20
)

func main() {
	flag.Parse()
	cfg, err := lac.FromFileJSON("config.json")

	rl.SetConfigFlags(rl.FlagWindowResizable | rl.FlagWindowTopmost | rl.FlagWindowTransparent | rl.FlagWindowUndecorated)
	rl.InitWindow(int32(windowWidth), int32(windowHeight), "movement possession")
	rl.SetTargetFPS(int32(rl.GetMonitorRefreshRate(rl.GetCurrentMonitor())))

	kCopy := getKey(cfg, "z", "copy")
	kSend := getKey(cfg, "", "send")
	kClear := getKey(cfg, "c", "clear")
	kClearAndSend := getKey(cfg, "enter", "sendAndClear")

	connectionStatus := "???"
	t := twitch.NewClient(cfg.GetDString("flexcoral", "auth", "username"), "oauth:"+figureToken(cfg))
	t.OnConnect(func() {
		connectionStatus = "connected"
	})
	t.Join(cfg.GetDString("necauqua", "channelName"))
	go func() {
		connectionStatus = "wait"
		err := t.Connect()
		if err != nil {
			connectionStatus = err.Error()
		}
	}()

	states := []state{{T: time.Now()}}

	lastActionTime := time.Time{}
	lastActionText := ""

	lastAutoSendTime := time.Time{}
	lastAutoSendContent := ""
	autoSendEnabled := false

	for !rl.WindowShouldClose() {
		if rl.IsKeyPressed(rl.KeyQ) {
			break
		}
		if rl.IsWindowResized() {
			windowWidth = int32(rl.GetRenderWidth())
			windowHeight = int32(rl.GetRenderHeight())
		}
		if rl.IsKeyPressed(rl.KeyMinus) {
			autoSendEnabled = !autoSendEnabled
		}
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			tresW := int32(1920)
			tresH := int32(1080)
			mp := rl.GetMousePosition()
			windowWidth = int32(rl.GetRenderWidth())
			windowHeight = int32(rl.GetRenderHeight())
			stx := int32((float64(mp.X) / float64(windowWidth)) * float64(tresW))
			sty := int32((float64(mp.Y) / float64(windowHeight)) * float64(tresH))
			// fmt.Println(windowWidth, windowHeight, mp, stx, sty)
			t.Say(cfg.GetDString("necauqua", "channelName"), fmt.Sprintf("+mouse:%d:%d", stx-(tresW/2), sty-(tresH/2)))
		}
		if (time.Since(lastAutoSendTime) > time.Second*5 && autoSendEnabled) || (rl.IsKeyPressed(kSend) && autoSendEnabled) {
			msg := statesToCommand(states)
			if msg != "" {
				if msg == lastAutoSendContent {
					msg += " I am not spamming"
				}
				t.Say(cfg.GetDString("necauqua", "channelName"), statesToCommand(states))
				lastAutoSendTime = time.Now()
				states = []state{{T: time.Now()}}
				lastActionText = "autosent"
				lastActionTime = time.Now()
			}
		}
		if rl.IsKeyPressed(kClear) {
			states = []state{{T: time.Now()}}
			lastActionText = "cleared"
			lastActionTime = time.Now()
		}
		if rl.IsKeyPressed(kSend) {
			msg := statesToCommand(states)
			if msg != "" {
				t.Say(cfg.GetDString("necauqua", "channelName"), statesToCommand(states))
			}
			lastActionText = "sent"
			lastActionTime = time.Now()
		}
		if rl.IsKeyPressed(kClearAndSend) {
			msg := statesToCommand(states)
			if msg != "" {
				t.Say(cfg.GetDString("necauqua", "channelName"), statesToCommand(states))
			}
			states = []state{{T: time.Now()}}
			lastActionText = "sent+cleared"
			lastActionTime = time.Now()
		}
		if rl.IsKeyPressed(kCopy) {
			msg := statesToCommand(states)
			rl.SetClipboardText(msg)
			lastActionText = "copied"
			lastActionTime = time.Now()
		}

		latestState := states[len(states)-1]
		if (rl.IsKeyDown(rl.KeyW) || rl.IsKeyDown(rl.KeySpace)) != latestState.W ||
			rl.IsKeyDown(rl.KeyA) != latestState.A ||
			rl.IsKeyDown(rl.KeyS) != latestState.S ||
			rl.IsKeyDown(rl.KeyD) != latestState.D {

			states = append(states, state{
				T: time.Now(),
				W: rl.IsKeyDown(rl.KeyW) || rl.IsKeyDown(rl.KeySpace),
				A: rl.IsKeyDown(rl.KeyA),
				S: rl.IsKeyDown(rl.KeyS),
				D: rl.IsKeyDown(rl.KeyD),
			})
		}

		rl.BeginDrawing()
		{
			rl.ClearBackground(color.RGBA{})
			rl.DrawRectangleLines(1, 1, windowWidth-1, windowHeight-1, rl.Red)
			rl.DrawRectangleLines(2, 2, windowWidth-4, windowHeight-4, rl.Red)
			// rl.DrawRectangleLines(3, 3, windowWidth-6, windowHeight-6, rl.Red)
			line := int32(2)
			if err != nil {
				rl.DrawText(fmt.Sprintf("Error: %v", err), 2, line, fontSize, rl.White)
				line += fontSize
			}
			rl.DrawText(fmt.Sprintf("Movement possession [twitch %s] [autosend %v]", connectionStatus, autoSendEnabled), 2, line, fontSize, rl.White)
			line += fontSize
			rl.DrawText(statesToCommand(states), 2, line, fontSize, rl.White)
			line += fontSize

			lam := uint8(max(int64(0), int64(254-(time.Since(lastActionTime).Milliseconds()/4))))
			if lam != 0 {
				rl.DrawText(lastActionText, 2, line, fontSize, color.RGBA{R: 255, G: 255, B: 255, A: lam})
				line += fontSize
			}
		}
		rl.EndDrawing()
	}
	t.Disconnect()
	rl.CloseWindow()
}

type state struct {
	W bool
	A bool
	S bool
	D bool
	T time.Time
}

func statesToCommand(states []state) string {
	ret := ""
	for i, v := range states {
		if i+1 >= len(states) {
			break
		}
		dur := strconv.Itoa(int(states[i+1].T.Sub(v.T).Milliseconds()))
		gd := []string{}
		if v.W {
			gd = append(gd, "+u"+dur)
		}
		if v.S {
			gd = append(gd, "+d"+dur)
		}
		if v.A {
			gd = append(gd, "+l"+dur)
		}
		if v.D {
			gd = append(gd, "+r"+dur)
		}
		if len(gd) == 0 {
			if ret != "" {
				ret += " +w" + dur
			}
		} else if len(gd) == 1 {
			ret += " " + gd[0]
		} else {
			ret += " +g:\"" + strings.Join(gd, " | ") + "\""
		}
	}
	return strings.TrimSpace(ret)
}

func getKey(cfg lac.Conf, dflt, name string) int32 {
	uk := strings.ToLower(cfg.GetDString(dflt, "keys", name))
	if uk == "" {
		return 0
	}
	k, ok := keyDefs[uk]
	if !ok {
		rl.TraceLog(rl.LogError, "key named %s is not valid", name)
		return keyDefs[strings.ToLower(dflt)]
	}
	return k
}

func figureToken(cfg lac.Conf) string {
	tokenBytes, err := os.ReadFile("token.txt")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			token := obtainToken(cfg)
			err := os.WriteFile("token.txt", []byte(token), 0644)
			if err != nil {
				fmt.Printf("Error saving token: %s", err.Error())
			}
			return token
		}
		must(err)
	}
	return string(tokenBytes)
}

func obtainToken(cfg lac.Conf) string {
	vals := url.Values{
		"client_id":     []string{cfg.GetDString("NOCLIENTID", "auth", "client_id")},
		"redirect_uri":  []string{cfg.GetDString("NOREDIRECT", "auth", "redirect_uri")},
		"response_type": []string{"token"},
		"scope":         []string{"user:write:chat user:read:chat chat:edit chat:read"},
	}

	fmt.Printf("Navigate to https://id.twitch.tv/oauth2/authorize?%s", vals.Encode())

	var gotToken string

	srv := http.Server{
		Addr: "0.0.0.0:50291",
	}
	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.URL.Query().Get("access_token")
		w.WriteHeader(200)
		w.Write([]byte("got token"))
		srv.Close()
	})
	fmt.Printf("Listening for credentials on %q", srv.Addr)
	srv.ListenAndServe()

	fmt.Println("got stuff")

	return gotToken
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func noerr[T any](ret T, err error) T {
	must(err)
	return ret
}

func noerr2[T, T2 any](ret T, ret2 T2, err error) (T, T2) {
	must(err)
	return ret, ret2
}
