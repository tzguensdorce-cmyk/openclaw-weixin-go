// Author: jtai团队（曾能混&tang先森） <jwhna1@gmil.com>
// Official Site: https://jtai.cc
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jwhna1/openclaw-weixin-go/client"
	"github.com/jwhna1/openclaw-weixin-go/store"
	"github.com/mdp/qrterminal/v3"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "login":
		must(runLogin(os.Args[2:]))
	case "whoami":
		must(runWhoAmI(os.Args[2:]))
	case "poll":
		must(runPoll(os.Args[2:]))
	case "send":
		must(runSend(os.Args[2:]))
	case "logout":
		must(runLogout(os.Args[2:]))
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`openclaw-weixin-go

Usage:
  openclaw-weixin-go login  --data-dir ./data
  openclaw-weixin-go whoami --data-dir ./data
  openclaw-weixin-go poll   --data-dir ./data
  openclaw-weixin-go send   --data-dir ./data --to wxid_xxx --text "hello"
  openclaw-weixin-go logout --data-dir ./data`)
}

func must(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func newStore(baseDir string) *store.FileStore {
	return store.NewFileStore(baseDir)
}

func newClient(baseURL string) *client.Client {
	return client.New(client.Options{
		BaseURL:        baseURL,
		ClientIDPrefix: "openclaw_weixin_go",
	})
}

func runLogin(args []string) error {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	dataDir := fs.String("data-dir", ".", "directory used to store account.json and cursors")
	baseURL := fs.String("base-url", client.DefaultBaseURL, "iLink API base URL")
	pollInterval := fs.Duration("poll-interval", 1500*time.Millisecond, "QR polling interval")
	if err := fs.Parse(args); err != nil {
		return err
	}

	st := newStore(*dataDir)
	cli := newClient(*baseURL)
	ctx := context.Background()

	qrResp, err := cli.FetchQRCode(ctx)
	if err != nil {
		return err
	}

	fmt.Println("OpenClaw Weixin Go CLI Login")
	fmt.Println()
	renderTerminalQRCode(qrResp.QRUrl)
	fmt.Println()
	fmt.Println("QR URL:")
	fmt.Println(qrResp.QRUrl)
	fmt.Println()
	fmt.Printf("QR Code Token: %s\n", qrResp.QRCode)
	fmt.Printf("Store Directory: %s\n", st.Dir())
	fmt.Println("Please scan the QR code with WeChat, confirm on your phone, and wait for login to complete...")
	fmt.Println("Status: waiting for scan")

	lastStatus := ""
	for {
		statusResp, err := cli.PollQRStatus(ctx, qrResp.QRCode, *baseURL)
		if err != nil {
			return err
		}
		if statusResp.Status != "" && statusResp.Status != lastStatus {
			fmt.Printf("Status: %s\n", describeQRStatus(statusResp.Status))
			lastStatus = statusResp.Status
		}
		if statusResp.ErrCode == 0 && statusResp.BotToken != "" {
			acct := &store.Account{
				Token:     statusResp.BotToken,
				BaseURL:   statusResp.BaseURL,
				AccountID: statusResp.AccountID,
			}
			if acct.BaseURL == "" {
				acct.BaseURL = *baseURL
			}
			if err := st.SaveAccount(acct); err != nil {
				return err
			}
			fmt.Println("Status: login confirmed")
			fmt.Println("Login succeeded.")
			fmt.Printf("Saved account to: %s\n", st.Dir())
			return nil
		}

		switch statusResp.Status {
		case "cancel", "cancelled":
			return fmt.Errorf("login canceled by user")
		case "expired", "expire":
			return fmt.Errorf("QR code expired, please run login again")
		}

		time.Sleep(*pollInterval)
	}
}

func renderTerminalQRCode(qrURL string) {
	if strings.TrimSpace(qrURL) == "" {
		return
	}
	config := qrterminal.Config{
		Level:          qrterminal.L,
		Writer:         os.Stdout,
		HalfBlocks:     true,
		BlackChar:      qrterminal.BLACK_BLACK,
		BlackWhiteChar: qrterminal.BLACK_WHITE,
		WhiteChar:      qrterminal.WHITE_WHITE,
		WhiteBlackChar: qrterminal.WHITE_BLACK,
		QuietZone:      1,
	}
	qrterminal.GenerateWithConfig(qrURL, config)
}

func describeQRStatus(status string) string {
	switch status {
	case "wait":
		return "waiting for scan"
	case "scan", "scanned":
		return "scanned, please confirm on your phone"
	case "success", "confirmed":
		return "login confirmed"
	case "cancel", "cancelled":
		return "login canceled"
	case "expired", "expire":
		return "QR code expired"
	default:
		if strings.TrimSpace(status) == "" {
			return "waiting for scan"
		}
		return status
	}
}

func runWhoAmI(args []string) error {
	fs := flag.NewFlagSet("whoami", flag.ContinueOnError)
	dataDir := fs.String("data-dir", ".", "directory used to store account.json and cursors")
	if err := fs.Parse(args); err != nil {
		return err
	}

	acct, err := newStore(*dataDir).LoadAccount()
	if err != nil {
		return err
	}
	if acct == nil {
		return fmt.Errorf("no local account found under %s", newStore(*dataDir).Dir())
	}
	body, err := json.MarshalIndent(acct, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return nil
}

func runPoll(args []string) error {
	fs := flag.NewFlagSet("poll", flag.ContinueOnError)
	dataDir := fs.String("data-dir", ".", "directory used to store account.json and cursors")
	timeoutMS := fs.Int("timeout-ms", client.DefaultGetUpdatesTimeout, "long-poll timeout in milliseconds")
	if err := fs.Parse(args); err != nil {
		return err
	}

	st := newStore(*dataDir)
	acct, err := st.LoadAccount()
	if err != nil {
		return err
	}
	if acct == nil || acct.Token == "" {
		return fmt.Errorf("not logged in")
	}
	cursor, err := st.LoadSyncCursor()
	if err != nil {
		return err
	}

	cli := newClient(acct.BaseURL)
	resp, err := cli.GetUpdates(context.Background(), acct.Token, cursor, *timeoutMS)
	if err != nil {
		return err
	}
	if resp.SyncBuf != "" && resp.SyncBuf != cursor {
		if err := st.SaveSyncCursor(resp.SyncBuf); err != nil {
			return err
		}
	}

	body, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return nil
}

func runSend(args []string) error {
	fs := flag.NewFlagSet("send", flag.ContinueOnError)
	dataDir := fs.String("data-dir", ".", "directory used to store account.json and cursors")
	toUserID := fs.String("to", "", "receiver user id")
	text := fs.String("text", "", "text content to send")
	contextToken := fs.String("context-token", "", "optional context token")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*toUserID) == "" {
		return fmt.Errorf("--to is required")
	}
	if strings.TrimSpace(*text) == "" {
		return fmt.Errorf("--text is required")
	}

	acct, err := newStore(*dataDir).LoadAccount()
	if err != nil {
		return err
	}
	if acct == nil || acct.Token == "" {
		return fmt.Errorf("not logged in")
	}

	_, err = newClient(acct.BaseURL).SendTextMessage(context.Background(), acct.Token, *toUserID, *text, *contextToken)
	if err != nil {
		return err
	}
	fmt.Println("Message sent.")
	return nil
}

func runLogout(args []string) error {
	fs := flag.NewFlagSet("logout", flag.ContinueOnError)
	dataDir := fs.String("data-dir", ".", "directory used to store account.json and cursors")
	if err := fs.Parse(args); err != nil {
		return err
	}

	st := newStore(*dataDir)
	if err := st.ClearAccount(); err != nil {
		return err
	}
	fmt.Printf("Removed local account from %s\n", st.Dir())
	return nil
}
