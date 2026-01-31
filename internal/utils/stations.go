package utils 

import "strings"

// StationOption represents a train station with its code and display name
type StationOption struct {
	Code        string
	DisplayName string
}

// PopularStations contains a hardcoded list of popular Russian train stations
// Ordered by usage frequency (most popular first)
var PopularStations = []StationOption{
	// Самые популярные станции (топ-5)
	{Code: "s9634302", DisplayName: "Красный Котельщик"},
	{Code: "s9613483", DisplayName: "Старый вокзал"},
	{Code: "s9613171", DisplayName: "Новый вокзал"},
	{Code: "s9612913", DisplayName: "Ростов-Главный"},
	{Code: "s9612913", DisplayName: "Ростов-Пригородный"},

	// Ростов-на-Дону - остальные станции
	{Code: "s9612913", DisplayName: "Ростов-на-Дону"},
	{Code: "s9612914", DisplayName: "Ростов-Товарный"},

	// Промежуточные станции Таганрог → Ростов
	{Code: "s9613486", DisplayName: "Мержаново"},
	{Code: "s9613487", DisplayName: "Матвеев Курган"},
	{Code: "s9613488", DisplayName: "Куйбышево"},
	{Code: "s9613489", DisplayName: "Синявка"},
	{Code: "s9613490", DisplayName: "Чалтырь"},
	{Code: "s9613491", DisplayName: "Большие Салы"},
	{Code: "s9613492", DisplayName: "Батайск"},
	{Code: "s9613493", DisplayName: "Азов"},
	{Code: "s9613255", DisplayName: "Бессергеновка"},
	{Code: "s9613383", DisplayName: "1283км"},

	// Other Southern Russia
	{Code: "s9607404", DisplayName: "Краснодар"},
	{Code: "s9623547", DisplayName: "Анапа"},
	{Code: "s9607398", DisplayName: "Сочи"},
	{Code: "s9635385", DisplayName: "Адлер"},
	{Code: "s9635145", DisplayName: "Новороссийск"},
	{Code: "s9620770", DisplayName: "Волгоград"},
	{Code: "s9635134", DisplayName: "Туапсе"},

	// Moscow region
	{Code: "s2000002", DisplayName: "Москва (Курский вокзал)"},
	{Code: "s2000006", DisplayName: "Москва (Казанский вокзал)"},
	{Code: "s2000003", DisplayName: "Москва (Ярославский вокзал)"},
	{Code: "s2000004", DisplayName: "Москва (Ленинградский вокзал)"},
	{Code: "s2000005", DisplayName: "Москва (Павелецкий вокзал)"},
	{Code: "s2000001", DisplayName: "Москва (Киевский вокзал)"},
	{Code: "s2000007", DisplayName: "Москва (Белорусский вокзал)"},

	// Saint Petersburg
	{Code: "s2004001", DisplayName: "Санкт-Петербург (Московский вокзал)"},
	{Code: "s2004006", DisplayName: "Санкт-Петербург (Витебский вокзал)"},
	{Code: "s2004003", DisplayName: "Санкт-Петербург (Ладожский вокзал)"},
	{Code: "s2004004", DisplayName: "Санкт-Петербург (Финляндский вокзал)"},

	// Volga region
	{Code: "s9610171", DisplayName: "Казань"},
	{Code: "s9623443", DisplayName: "Нижний Новгород"},
	{Code: "s9608105", DisplayName: "Самара"},
	{Code: "s9623290", DisplayName: "Саратов"},
	{Code: "s9623214", DisplayName: "Уфа"},
	{Code: "s9608191", DisplayName: "Пермь"},

	// Ural
	{Code: "s9607693", DisplayName: "Екатеринбург"},
	{Code: "s9607795", DisplayName: "Челябинск"},
	{Code: "s9623371", DisplayName: "Тюмень"},

	// Siberia
	{Code: "s9607120", DisplayName: "Новосибирск"},
	{Code: "s9607077", DisplayName: "Омск"},
	{Code: "s9623307", DisplayName: "Красноярск"},
	{Code: "s9635387", DisplayName: "Иркутск"},
	{Code: "s9635427", DisplayName: "Владивосток"},
	{Code: "s9635386", DisplayName: "Хабаровск"},

	// Central Russia
	{Code: "s9612893", DisplayName: "Воронеж"},
	{Code: "s9613016", DisplayName: "Белгород"},
	{Code: "s9607881", DisplayName: "Тула"},
	{Code: "s9623147", DisplayName: "Рязань"},
	{Code: "s9623210", DisplayName: "Тамбов"},
	{Code: "s9623352", DisplayName: "Липецк"},
	{Code: "s9623254", DisplayName: "Курск"},
	{Code: "s9623269", DisplayName: "Орёл"},

	// North Caucasus
	{Code: "s9635342", DisplayName: "Минеральные Воды"},
	{Code: "s9635329", DisplayName: "Пятигорск"},
	{Code: "s9635331", DisplayName: "Кисловодск"},
	{Code: "s9635324", DisplayName: "Ессентуки"},
	{Code: "s9635373", DisplayName: "Грозный"},
	{Code: "s9635346", DisplayName: "Махачкала"},

	// Northwest
	{Code: "s9623204", DisplayName: "Мурманск"},
	{Code: "s9623278", DisplayName: "Петрозаводск"},
	{Code: "s9623421", DisplayName: "Псков"},
	{Code: "s9623434", DisplayName: "Великий Новгород"},
	{Code: "s9623362", DisplayName: "Архангельск"},
}

// SearchStations searches for stations by query string
// Returns up to 10 matching stations
func SearchStations(query string) []StationOption {
	if query == "" {
		// Return top 10 most popular stations
		if len(PopularStations) > 10 {
			return PopularStations[:10]
		}
		return PopularStations
	}

	query = strings.ToLower(strings.TrimSpace(query))
	results := []StationOption{}

	// Search for matches (case-insensitive substring match)
	for _, station := range PopularStations {
		if strings.Contains(strings.ToLower(station.DisplayName), query) {
			results = append(results, station)
		}
	}

	// Limit to 10 results
	if len(results) > 10 {
		return results[:10]
	}

	return results
}

// GetStationByCode returns station by code, or empty StationOption if not found
func GetStationByCode(code string) (StationOption, bool) {
	for _, station := range PopularStations {
		if station.Code == code {
			return station, true
		}
	}
	return StationOption{}, false
}

// GetStationByIndex returns station by index in PopularStations slice
func GetStationByIndex(index int) (StationOption, bool) {
	if index < 0 || index >= len(PopularStations) {
		return StationOption{}, false
	}
	return PopularStations[index], true
}
