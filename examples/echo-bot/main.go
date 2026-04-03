// Author: jtai团队（曾能混&tang先森） <jwhna1@gmil.com>
// Official Site: https://jtai.cc
package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/jwhna1/openclaw-weixin-go/client"
	"github.com/jwhna1/openclaw-weixin-go/store"
)

func main() {
	dataDir := flag.String("data-dir", ".", "directory used to store account.json and cursors")
	flag.Parse()

	st := store.NewFileStore(*dataDir)
	acct, err := st.LoadAccount()
	if err != nil {
		panic(err)
	}
	if acct == nil || acct.Token == "" {
		panic("please login first with openclaw-weixin-go login")
	}

	ctxTokens, err := st.LoadContextTokens()
	if err != nil {
		panic(err)
	}
	cursor, err := st.LoadSyncCursor()
	if err != nil {
		panic(err)
	}

	cli := client.New(client.Options{
		BaseURL:        acct.BaseURL,
		ClientIDPrefix: "echo_bot",
	})

	fmt.Println("echo bot started")
	for {
		resp, err := cli.GetUpdates(context.Background(), acct.Token, cursor, client.DefaultGetUpdatesTimeout)
		if err != nil {
			panic(err)
		}
		if resp.SyncBuf != "" && resp.SyncBuf != cursor {
			cursor = resp.SyncBuf
			if err := st.SaveSyncCursor(cursor); err != nil {
				panic(err)
			}
		}
		for _, msg := range resp.MessageList {
			if msg.MessageType == client.MessageTypeBot {
				continue
			}
			var parts []string
			for _, item := range msg.ItemList {
				if item.TextItem != nil && item.TextItem.Text != "" {
					parts = append(parts, item.TextItem.Text)
				}
			}
			text := strings.TrimSpace(strings.Join(parts, ""))
			if text == "" || msg.FromUserID == "" {
				continue
			}
			if msg.ContextToken != "" {
				ctxTokens[msg.FromUserID] = msg.ContextToken
				if err := st.SaveContextTokens(ctxTokens); err != nil {
					panic(err)
				}
			}
			if _, err := cli.SendTextMessage(context.Background(), acct.Token, msg.FromUserID, "echo: "+text, ctxTokens[msg.FromUserID]); err != nil {
				panic(err)
			}
			fmt.Printf("replied to %s: %s\n", msg.FromUserID, text)
		}
	}
}
