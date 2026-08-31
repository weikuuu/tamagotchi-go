// Package phrases holds Elysia's speech-bubble lines, organized by the game
// state that should trigger them so they're easy to wire up from the UI
// code (see ForState, ForAction, Night, Idle, LongAbsence, Petted below).
package phrases

import (
	"fmt"
	"math/rand"

	"elygochi/internal/pet"
)

// Category names a group of lines tied to one game situation.
type Category string

const (
	Hungry         Category = "hungry"          // low Hunger
	AfterEat       Category = "after_eat"       // just fed
	Sleepy         Category = "sleepy"          // low Energy
	WakeUp         Category = "wake_up"         // just rested
	WantPlay       Category = "want_play"       // bored, wants to play
	AfterPlay      Category = "after_play"      // just played
	Happy          Category = "happy"           // high Happiness
	Sad            Category = "sad"             // low Happiness
	Upset          Category = "upset"           // neglected / annoyed
	LongAbsence    Category = "long_absence"    // hasn't been opened in a long time
	Random         Category = "random"          // ambient filler, any time
	Dirty          Category = "dirty"           // needs grooming (low Cleanliness)
	Music          Category = "music"           // ambient reaction to music playing
	Sick           Category = "sick"            // MoodSick
	HighFriendship Category = "high_friendship" // everything's great, high bond
	Night          Category = "night"           // late/early local hours
	Idle           Category = "idle"            // no interaction for a while
	Petted         Category = "petted"          // clicked/petted on the overlay
	Ultra          Category = "ultra"           // rare "feels alive" easter eggs
	Birthday       Category = "birthday"        // special event
	NewYear        Category = "new_year"        // special event
	Streak         Category = "streak"          // long run without neglect
	Flee           Category = "flee"            // darting away from the cursor
	FleeToggleOn   Category = "flee_toggle_on"  // "chase me" mode just turned on
	FleeToggleOff  Category = "flee_toggle_off" // "chase me" mode just turned off
	MiniGameGood   Category = "mini_game_good"  // caught most of the hearts
	MiniGameOk     Category = "mini_game_ok"    // caught a few hearts
)

var byCategory = map[Category][]string{
	Hungry: {
		"Ммм… кажется, мой желудок решил напомнить о себе~",
		"Эй, а где моя вкусняшка?",
		"Я вовсе не капризничаю… я просто очень-очень голодная!",
		"Как насчёт чего-нибудь вкусненького?",
		"Мне кажется, я заслужила маленькое угощение~",
		"Пожалуйста, покорми меня!",
		"Еда! Еда! Еда!",
		"Я начинаю смотреть на всё вокруг как на еду…",
		"Ты ведь не собираешься оставить меня голодной?",
		"Мой желудок поёт серенаду. Очень грустную.",
		"Мне нужно срочно что-нибудь сладенькое!",
		"А можно мне ещё кусочек? Ну пожа-а-алуйста~",
		"Вот сейчас самое время для вкусного перерыва!",
		"Я готова к великому пиру!",
		"Если меня покормить, я стану ещё прекраснее. Наверное.",
	},
	AfterEat: {
		"Ммм~ Какое счастье!",
		"Вот теперь я снова полна энергии!",
		"Спасибо! Это было чудесно~",
		"Мой животик счастлив, а значит, счастлива и я!",
		"Ещё чуть-чуть — и я бы стала воздушным шариком~",
		"Как вкусно! Ты умеешь меня баловать.",
		"Теперь можно снова покорять мир!",
		"Это было прекрасно. Повторим когда-нибудь?",
		"Ням! Идеально~",
		"Я чувствую себя настоящей принцессой!",
	},
	Sleepy: {
		"Уууу… мои глазки становятся тяжёлыми…",
		"Кажется, пора отправляться в страну снов~",
		"Я ещё совсем не хочу спать… з-зевок…",
		"Можно я немножко отдохну?",
		"Мои батарейки почти разрядились…",
		"Ещё пять минут… пожалуйста…",
		"Мне снится что-то прекрасное ещё до того, как я уснула~",
		"Я так устала… обними… э-э, укрой меня одеялом!",
		"Завтра продолжим наши приключения.",
		"Доброй ночи, мой дорогой друг~",
		"Шшш… Элизия заряжается.",
		"Я ухожу в царство сладких снов~",
	},
	WakeUp: {
		"Доброе утро! Разве мир сегодня не прекрасен?",
		"Я проснулась! Соскучился?",
		"Какой чудесный день для новых приключений!",
		"Я готова! Ну… почти готова. Ещё немножко потянусь~",
		"Доброе утро, дорогой!",
		"Солнце уже встало, а я всё ещё прекрасна. Какая неожиданность~",
		"Что будем делать сегодня?",
		"Новый день — новая возможность стать ещё счастливее!",
		"Я прекрасно спала. А ты?",
		"Ну же! Нас ждёт целый день!",
	},
	WantPlay: {
		"Поиграем? Поиграем! Поиграем!!",
		"Мне скучно… спасёшь меня?",
		"А давай немного повеселимся~",
		"Я хочу играть!",
		"Кто-нибудь сказал «игра»?",
		"Ну же, придумай что-нибудь интересное!",
		"Я готова бросить тебе вызов!",
		"Хочешь проверить, кто из нас лучше?",
		"Мне нужно немного развлечений~",
		"Давай устроим маленькое приключение!",
		"Скука — мой главный враг!",
		"Я придумала игру! Правда, пока не придумала какую…",
	},
	AfterPlay: {
		"Ха-ха! Было весело!",
		"Ещё! Ещё! Ещё!",
		"Это было прекрасно~",
		"Ты неплохо справилась!",
		"Мне понравилось! Повторим?",
		"Ух ты… я даже немного устала.",
		"Какое чудесное развлечение!",
		"Победа! Ну… почти победа.",
		"С тобой никогда не бывает скучно~",
		"Это определённо стоило того!",
	},
	Happy: {
		"Сегодня я особенно счастлива~",
		"Разве жизнь не прекрасна?",
		"У меня такое хорошее настроение!",
		"Мне хочется улыбаться весь день!",
		"Я чувствую себя сияющей звездой~",
		"Сегодня всё просто идеально.",
		"Хи-хи~ Мне хорошо.",
		"Я счастлива, что ты здесь.",
		"Моё сердце переполнено радостью!",
		"Кажется, сегодня будет чудесный день.",
	},
	Sad: {
		"Что-то мне сегодня совсем не весело…",
		"Мне немного одиноко.",
		"Ты можешь побыть со мной?",
		"Я не знаю почему, но настроение куда-то убежало…",
		"Можно немного внимания?..",
		"Сегодня мне хочется просто посидеть спокойно.",
		"Я скоро повеселею. Наверное.",
		"Мне не хватает чего-то хорошего…",
		"Эй… не уходи пока.",
		"Может, ты сможешь меня развеселить?",
	},
	Upset: {
		"Эй! Я вообще-то жду внимания!",
		"Хм! Я это запомню.",
		"Ты меня совсем забросил!",
		"Я возмущена!",
		"Ну и ну… какое безобразие.",
		"Я требую объяснений!",
		"Это не очень-то красиво с твоей стороны.",
		"Хм-м. Я на тебя смотрю.",
		"Не заставляй меня сердиться~",
		"Я могу быть очень убедительной, знаешь?",
	},
	LongAbsence: {
		"Ты где была?!",
		"Наконец-то! Я уже начала скучать!",
		"Ты совсем про меня забыла?",
		"Я ждала тебя~",
		"Ну наконец-то мой любимый человек появился!",
		"Я уже думала, ты потеряла дорогу ко мне.",
		"Смотри, кто решила вернуться~",
		"Я не сержусь… почти.",
		"Ты заставила меня ждать.",
		"Я рада тебя видеть! Но вообще-то ты опоздала.",
	},
	Random: {
		"Ты когда-нибудь задумывалась, почему облака такие красивые?",
		"Мне кажется, сегодня воздух пахнет приключениями~",
		"Я только что придумала кое-что гениальное.",
		"А ты знала, что я прекрасна? Конечно, знала.",
		"Мне нечем заняться… придумай что-нибудь!",
		"Интересно, что сейчас происходит в мире?",
		"Иногда так приятно просто ничего не делать.",
		"Я сегодня особенно хороша, не находишь?",
		"Хи-хи~ Я кое-что задумала.",
		"Не смотри на меня так. Я ничего не делала!",
		"Мне кажется, за мной сегодня наблюдают…",
		"Ты ведь знаешь, что я тебе доверяю?",
		"Как думаешь, кем я стану, когда вырасту?",
		"Я хочу увидеть что-нибудь красивое!",
		"А давай сегодня будем счастливы без причины?",
		"Мир такой большой… а мы сидим здесь.",
		"У меня появилась блестящая идея!",
		"Пожалуй, я сегодня буду особенно очаровательной.",
		"Иногда мне кажется, что я живу внутри маленькой сказки.",
		"Хм… ты тоже это слышала?",
		"Я уверена, что сегодня произойдёт что-нибудь интересное.",
	},
	Dirty: {
		"Кажется, мне пора привести себя в порядок~",
		"Ой… я немножко испачкалась.",
		"Купание! Отличная идея!",
		"Нельзя же оставаться такой растрёпанной.",
		"Мне нужна маленькая помощь~",
		"После этого я снова буду сиять!",
		"Фух… теперь гораздо лучше.",
		"Вот! Снова прекрасна.",
	},
	Sick: {
		"Мне что-то нехорошо…",
		"Кажется, сегодня я не в лучшей форме.",
		"Можно мне немного заботы?",
		"Я хочу просто отдохнуть.",
		"Мне нужно восстановить силы.",
		"Не волнуйся, я скоро поправлюсь.",
		"Сегодня я немного слабее обычного…",
		"Давай побережём меня сегодня.",
	},
	HighFriendship: {
		"Ты знаешь, с тобой действительно весело.",
		"Я так рада, что именно ты обо мне заботишься~",
		"Кажется, мы стали настоящей командой.",
		"Ты меня совсем избаловала.",
		"Я всегда рада тебя видеть!",
		"Ты стала важной частью моих маленьких приключений.",
		"Хи-хи… мне очень хорошо рядом с тобой.",
		"Спасибо, что не забываешь обо мне.",
		"С тобой даже обычный день становится особенным.",
		"Давай будем друзьями ещё очень-очень долго~",
	},
	Night: {
		"Почему ты ещё не спишь?",
		"Ночь такая тихая…",
		"Звёзды сегодня особенно красивые.",
		"Давай посмотрим на небо ещё немного.",
		"Мне нравится ночь. В ней есть что-то волшебное.",
		"Пора отдыхать, иначе завтра будем сонными~",
		"Тс-с… кажется, весь мир уже спит.",
		"Спокойной ночи. До завтра~",
	},
	Idle: {
		"…",
		"Эй.",
		"Ты там живой?",
		"Мне скучно.",
		"Я всё ещё здесь, между прочим.",
		"Ну ла-а-адно… я подожду.",
	},
	Petted: {
		"Ня~!",
		"Ещё, ещё!",
		"Мне нравится!",
		"Хи-хи, щекотно!",
	},
	Ultra: {
		"Пс-с… только никому не говори, но ты моя любимица~",
		"Я сегодня видела сон. Кажется, ты там тоже была.",
		"Знаешь… иногда мне просто хочется, чтобы этот момент длился подольше.",
		"Если ты улыбаешься, значит, всё хорошо. Наверное~",
		"Эй, давай сегодня ничего не будем делать. Просто побудем здесь.",
		"Я кое-что поняла… мне нравится этот маленький мир.",
		"Хи-хи. Ты ведь тоже заметила, что я стала счастливее?",
		"Не переживай. Я здесь.",
		"Как думаешь… мы надолго вместе?",
		"Тогда решено! Завтра тоже будет прекрасным днём!",
	},
	Birthday: {
		"С днём рождения! Я весь день ждала, чтобы это сказать!",
		"Сегодня твой особенный день — я никак не могла об этом забыть!",
		"Ура! Пусть этот год будет для тебя самым лучшим!",
		"Я бы подарила тебе целый букет, будь у меня руки подлиннее~",
	},
	NewYear: {
		"Новый год! Новые приключения!",
		"Интересно, что принесёт нам этот год?",
		"Пусть впереди будет много счастливых дней~",
	},
	Streak: {
		"Ого! Ты становишься всё лучше!",
		"Я тобой горжусь~",
		"Вот это команда!",
	},
	Flee: {
		"Ай, не поймаешь!",
		"Пятнашки? Я в деле~",
		"Ускользаю! Хи-хи~",
		"Слишком медленно!",
		"А вот и не поймал!",
		"Куда я, туда и ты? Забавно~",
		"Догоняй, если сможешь!",
		"Уииии, погоня!",
	},
	FleeToggleOn: {
		"Ловишки! Попробуй меня поймать~",
		"Хи-хи, теперь я буду убегать!",
		"Игра в догонялки началась!",
		"Ну попробуй поймай меня~",
	},
	FleeToggleOff: {
		"Ладно, больше не буду убегать~",
		"Уф, устала бегать. Отдохнём?",
		"Хорошо, я снова смирная~",
		"На сегодня погоня окончена.",
	},
	MiniGameGood: {
		"Ты все сердечки поймала! Так и знала, что мы отличная команда~",
		"Вау! Ты просто чемпионка!",
		"Идеально! Мне ещё никто так быстро не помогал.",
		"С тобой даже ловить сердечки веселее обычного!",
	},
	MiniGameOk: {
		"Неплохо! В следующий раз поймаем ещё больше~",
		"Хи-хи, было весело, даже если не идеально.",
		"Ты старалась, а это самое важное!",
		"Ещё разок? Я в деле!",
	},
	Music: {
		"Какая приятная музыка~",
		"Мне нравится эта мелодия!",
		"Потанцуем под эту песню?",
		"У тебя отличный музыкальный вкус.",
		"Я уже подпеваю про себя~",
		"Хорошая песня для настроения.",
		"Обожаю, когда играет музыка.",
		"Сделай погромче, а?",
	},
}

// musicTrackReactions are format strings taking the artist's name, used
// when the currently playing track changes.
var musicTrackReactions = []string{
	"О, %s? Хороший выбор!",
	"Ты слушаешь %s? Мне нравится~",
	"%s — неплохо, неплохо.",
	"Опять %s! Кажется, это твой любимчик.",
}

// TrackReaction returns a line reacting to a newly-started track by artist.
func TrackReaction(artist string) string {
	if artist == "" {
		return Get(Music)
	}
	return fmt.Sprintf(musicTrackReactions[rand.Intn(len(musicTrackReactions))], artist)
}

// nameReactions are format strings taking the user's name, for ambient
// lines that address them by it.
var nameReactions = []string{
	"%s, привет!",
	"Как твои дела, %s?",
	"%s, я так рада, что ты рядом~",
	"Эй, %s~",
	"%s, ты сегодня особенно прекрасна!",
	"Знаешь, %s, мне с тобой очень хорошо.",
	"%s… мне нравится, как звучит твоё имя.",
	"Соскучилась по тебе, %s!",
}

// WithName returns a random line addressing the user by name. If name is
// empty, it falls back to a generic Random line.
func WithName(name string) string {
	if name == "" {
		return Get(Random)
	}
	return fmt.Sprintf(nameReactions[rand.Intn(len(nameReactions))], name)
}

// ultraChance is the probability that an ambient pick is replaced by a rare
// Ultra line, per the "1-3%" suggestion.
const ultraChance = 0.02

// highFriendshipThreshold is how high Affection needs to be for the
// occasional HighFriendship line to show up instead of a plain Happy one.
const highFriendshipThreshold = pet.Stat(60)

// Get returns a random line from the given category.
func Get(c Category) string {
	return pick(byCategory[c])
}

// shortMaxLen is the longest a line can be (in runes) to qualify for
// GetShort/ForStateShort/ForActionShort — tuned to what actually fits in
// the tiny desktop-overlay speech bubble, which is much smaller than the
// one in the main window.
const shortMaxLen = 34

// GetShort returns a random SHORT line from the given category, for
// display spots too small for the full-length lines (the desktop
// overlay's bubble). Falls back to the shortest available line in the
// category if none qualify.
func GetShort(c Category) string {
	return pickShort(byCategory[c])
}

// ForState picks an ambient line fitting the pet's current stats: mostly
// mood-driven, with a chance of a HighFriendship line when things are going
// especially well and a small chance of a rare Ultra line at any time.
func ForState(s pet.State) string {
	return forStateWith(s, Get)
}

// ForStateShort is ForState using only short lines; see GetShort.
func ForStateShort(s pet.State) string {
	return forStateWith(s, GetShort)
}

func forStateWith(s pet.State, get func(Category) string) string {
	if rand.Float64() < ultraChance {
		return get(Ultra)
	}
	if s.IsDirty() && rand.Float64() < 0.5 {
		return get(Dirty)
	}
	if s.Affection >= highFriendshipThreshold && rand.Float64() < 0.3 {
		return get(HighFriendship)
	}
	switch s.Mood() {
	case pet.MoodHappy:
		if rand.Float64() < 0.35 {
			return get(Random)
		}
		return get(Happy)
	case pet.MoodContent:
		return get(Random)
	case pet.MoodBored:
		return get(WantPlay)
	case pet.MoodSad:
		if rand.Float64() < 0.4 {
			return get(Upset)
		}
		return get(Sad)
	case pet.MoodHungry:
		return get(Hungry)
	case pet.MoodTired:
		return get(Sleepy)
	case pet.MoodSick:
		return get(Sick)
	default:
		return get(Random)
	}
}

// ForAction returns a reaction line for a named action
// ("feed", "play", "rest", or "wash").
func ForAction(action string, s pet.State) string {
	return forActionWith(action, s, Get)
}

// ForActionShort is ForAction using only short lines; see GetShort.
func ForActionShort(action string, s pet.State) string {
	return forActionWith(action, s, GetShort)
}

func forActionWith(action string, s pet.State, get func(Category) string) string {
	switch action {
	case "feed":
		return get(AfterEat)
	case "play":
		return get(AfterPlay)
	case "rest":
		return get(WakeUp)
	case "wash":
		return get(Dirty)
	default:
		return forStateWith(s, get)
	}
}

func pick(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return lines[rand.Intn(len(lines))]
}

// pickShort picks among lines no longer than shortMaxLen runes. If none
// qualify, it falls back to the single shortest line so callers always get
// something reasonably compact.
func pickShort(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	var candidates []string
	shortest := lines[0]
	for _, l := range lines {
		if len([]rune(l)) <= shortMaxLen {
			candidates = append(candidates, l)
		}
		if len([]rune(l)) < len([]rune(shortest)) {
			shortest = l
		}
	}
	if len(candidates) == 0 {
		return shortest
	}
	return candidates[rand.Intn(len(candidates))]
}
