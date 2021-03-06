package main

import (
	"fmt"
	"strings"
	"whapp-irc/whapp"
)

func (conn *Connection) alternativeReplayWhappMessageHandle(msg whapp.Message) error {
	var err error

	chat := conn.GetChatByID(msg.Chat.ID)
	if chat == nil {
		chat, err = conn.addChat(msg.Chat)
		if err != nil {
			return err
		}
	}

	if chat.HasMessageID(msg.ID.Serialized) {
		return nil // already handled
	}
	chat.AddMessageID(msg.ID.Serialized)

	lastTimestamp, found := conn.timestampMap.Get(chat.ID.String())
	if !found || msg.Timestamp > lastTimestamp {
		conn.timestampMap.Set(chat.ID.String(), msg.Timestamp)
		go conn.saveDatabaseEntry()
	}

	if msg.IsNotification {
		return nil
	}

	sender := formatContact(*msg.Sender, false)
	from := sender.SafeName()
	if msg.IsSentByMe {
		from = conn.nickname
	}

	var to string
	if chat.IsGroupChat || msg.IsSentByMe {
		to = chat.Identifier()
	} else {
		to = conn.nickname
	}

	if err := downloadAndStoreMedia(msg); err != nil {
		return err
	}

	message := getMessageBody(msg, chat.Participants, conn.me)
	for _, line := range strings.Split(message, "\n") {
		logMessage(msg.Time(), from, to, line)

		msg := fmt.Sprintf(
			"(%s) %s->%s: %s",
			msg.Time().Format("2006-01-02 15:04:05"),
			from,
			to,
			line,
		)
		str := formatPrivateMessage("replay", conn.nickname, msg)
		if err := conn.writeIRCNow(str); err != nil {
			return err
		}
	}

	return nil
}
