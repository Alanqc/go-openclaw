package discord

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/openclaw/openclaw-go/internal/config"
	"github.com/openclaw/openclaw-go/internal/routing"
)

// PreflightParams holds input for preflight.
type PreflightParams struct {
	Cfg             *config.Config
	DiscordCfg      *DiscordConfig
	AccountID       string
	BotUserID       string
	Data            *discordgo.MessageCreate
	DMEnabled       bool
	GroupDMEnabled  bool
	GuildEntries    map[string]GuildEntry
}

// DiscordConfig (simplified).
type DiscordConfig struct {
	AllowBots bool
	DMPolicy  string
}

// GuildEntry (simplified).
type GuildEntry struct {
	ID    string
	Slug  string
	Allow bool
}

// PreflightContext is the result of successful preflight.
type PreflightContext struct {
	Cfg               *config.Config
	DiscordCfg        *DiscordConfig
	AccountID         string
	Message           *discordgo.Message
	Author            *discordgo.User
	Data              *discordgo.MessageCreate
	IsGuildMessage  bool
	IsDirectMessage bool
	IsGroupDM       bool
	BaseText        string
	MessageText     string
	WasMentioned    bool
	GuildID         string
	ChannelID       string
	ChannelName     string
	Route           routing.ResolvedAgentRoute
	CommandAuthorized bool
}

// Preflight validates and prepares the message. Returns nil if message should be dropped.
func Preflight(p PreflightParams) *PreflightContext {
	msg := p.Data.Message
	if msg == nil {
		return nil
	}
	author := msg.Author
	if author == nil {
		return nil
	}

	if author.Bot {
		if p.BotUserID != "" && author.ID == p.BotUserID {
			return nil
		}
		if p.DiscordCfg == nil || !p.DiscordCfg.AllowBots {
			slog.Debug("discord: drop bot message")
			return nil
		}
	}

	isGuild := p.Data.GuildID != ""
	var isDM, isGroupDM bool
	if p.Data.GuildID == "" {
		if len(p.Data.Mentions) > 0 || msg.Type == discordgo.MessageTypeDefault {
			isDM = true
		}
	}

	if isGroupDM && !p.GroupDMEnabled {
		slog.Debug("discord: drop group dm (disabled)")
		return nil
	}
	if isDM && !p.DMEnabled {
		slog.Debug("discord: drop dm (disabled)")
		return nil
	}

	baseText := msg.Content
	messageText := baseText

	peerKind := routing.PeerChannel
	peerID := msg.ChannelID
	if isDM {
		peerKind = routing.PeerDM
		peerID = author.ID
	} else if isGroupDM {
		peerKind = routing.PeerGroup
	}

	route := routing.ResolveAgentRoute(routing.ResolveAgentRouteInput{
		Cfg:       p.Cfg,
		Channel:   "discord",
		AccountID: p.AccountID,
		Peer:      &routing.RoutePeer{Kind: peerKind, ID: peerID},
		GuildID:   p.Data.GuildID,
	})

	wasMentioned := false
	if isGuild && p.BotUserID != "" {
		for _, u := range msg.Mentions {
			if u.ID == p.BotUserID {
				wasMentioned = true
				break
			}
		}
	}
	if isDM {
		wasMentioned = true
	}

	channelName := msg.ChannelID
	if msg.ChannelID != "" && p.Data.GuildID != "" {
		channelName = msg.ChannelID
	}

	return &PreflightContext{
		Cfg:              p.Cfg,
		DiscordCfg:       p.DiscordCfg,
		AccountID:        p.AccountID,
		Message:          msg,
		Author:           author,
		Data:             p.Data,
		IsGuildMessage:   isGuild,
		IsDirectMessage:  isDM,
		IsGroupDM:        isGroupDM,
		BaseText:         baseText,
		MessageText:      messageText,
		WasMentioned:     wasMentioned,
		GuildID:          p.Data.GuildID,
		ChannelID:        msg.ChannelID,
		ChannelName:      channelName,
		Route:            route,
		CommandAuthorized: true,
	}
}
