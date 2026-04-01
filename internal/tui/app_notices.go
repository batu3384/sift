package tui

const (
	appNoticeTicks       = 14
	routeMessageTicks    = 12
	routeMessageLongTicks = 20
)

func (m *appModel) setNotice(message string) {
	if message == "" {
		m.noticeMsg = ""
		m.noticeTicks = 0
		return
	}
	m.noticeMsg = message
	m.noticeTicks = appNoticeTicks
}

func (m *appModel) clearNotice() {
	m.noticeMsg = ""
	m.noticeTicks = 0
}

func (m *appModel) tickNotices() {
	if m.noticeTicks > 0 {
		m.noticeTicks--
		if m.noticeTicks == 0 {
			m.noticeMsg = ""
		}
	}
	if m.uninstall.messageTicks > 0 {
		m.uninstall.messageTicks--
		if m.uninstall.messageTicks == 0 {
			m.uninstall.message = ""
		}
	}
	if m.protect.messageTicks > 0 {
		m.protect.messageTicks--
		if m.protect.messageTicks == 0 {
			m.protect.message = ""
		}
	}
}
