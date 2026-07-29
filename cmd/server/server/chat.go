package server

import (
	"sync"

	"emperror.dev/errors"
	"github.com/chuccp/go-agent-sdk/agent"
	"github.com/chuccp/go-ai-agent/cmd/server/entity"
	"github.com/chuccp/go-web-frame/util"
)

type Chat struct {
	chatManager *agent.ChatManager
	chatClient  *agent.ChatClient
	lock        sync.Mutex
}

func (c *Chat) HandleChat(message *entity.RevMessage) error {
	client, err := c.getChatClient(message.SessionId)
	if err != nil {
		return err
	}
	return client.SendText(message.Message)

}

func (c *Chat) HandleStop(message *entity.RevMessage) error {
	client, err := c.getChatClient(message.SessionId)
	if err != nil {
		return err
	}
	client.Stop()
	return nil

}
func (c *Chat) getChatClient(id string) (*agent.ChatClient, error) {
	if util.IsBlank(id) {
		return nil, errors.New("id is blank")
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.chatClient != nil {
		return c.chatClient, nil
	}
	c.chatClient = c.chatManager.GetChat(id)
	return c.chatClient, nil
}
func (c *Chat) Release() {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.chatClient != nil {
		c.chatClient.Close()
	}
}

func newChat(chatManager *agent.ChatManager) *Chat {
	return &Chat{chatManager: chatManager}
}
