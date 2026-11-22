package app

import "sync"

type trayText struct {
	connected, connecting, disconnecting, disconnected, failed string
	connect, disconnect                                        string
	server, subscription, mode, routing                        string
	auto, proxy, rule, global, direct                          string
	empty, show, quit                                          string
}

var trayTexts = map[string]trayText{
	"en": {
		connected: "Connected", connecting: "Connecting…", disconnecting: "Disconnecting…",
		disconnected: "Disconnected", failed: "Couldn't connect",
		connect: "Connect", disconnect: "Disconnect",
		server: "Server", subscription: "Subscription", mode: "Mode", routing: "Routing",
		auto: "Auto", proxy: "Proxy", rule: "Rules", global: "Global", direct: "Direct",
		empty: "Nothing here yet", show: "Show ctun", quit: "Quit ctun",
	},
	"ru": {
		connected: "Подключено", connecting: "Подключение…", disconnecting: "Отключение…",
		disconnected: "Отключено", failed: "Не удалось подключиться",
		connect: "Подключить", disconnect: "Отключить",
		server: "Сервер", subscription: "Подписка", mode: "Режим", routing: "Маршрутизация",
		auto: "Авто", proxy: "Прокси", rule: "Правила", global: "Глобально", direct: "Напрямую",
		empty: "Нет элементов", show: "Показать ctun", quit: "Выйти из ctun",
	},
}

var systemLanguageOnce = sync.OnceValue(systemLanguage)

func trayTextFor(locale string) trayText {
	if locale == "" {
		locale = systemLanguageOnce()
	}
	if text, ok := trayTexts[locale]; ok {
		return text
	}
	return trayTexts["en"]
}
