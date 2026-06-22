package common

// Standard date/time format constants.
const (
	// Machine-readable formats
	DateFormat     = "2006-01-02"
	TimeFormat     = "15:04"
	DateTimeFormat = "2006-01-02 15:04"

	// Display formats (user-friendly)
	DisplayDateFormat = "Jan 2, 2006"
	DisplayTimeFormat = "3:04 PM"
	DisplayDateTime   = "Jan 2, 2006 3:04 PM"

	// Extended display formats with timezone
	DisplayDateTimeWithTZ     = "Jan 2, 2006 3:04 PM MST"
	DisplayWeekdayDateTime    = "Mon, Jan 2, 2006 at 3:04 PM MST"
	DisplayTimeWithTZ         = "3:04 PM MST"
	DisplayWeekdayShort       = "Mon Jan 2, 3:04 PM"
	DisplayWeekdayShortWithTZ = "Mon Jan 2, 3:04 PM MST"

	// Full weekday formats with year
	DisplayWeekdayFull       = "Mon Jan 2, 2006 3:04 PM"
	DisplayWeekdayFullWithTZ = "Mon Jan 2, 2006 3:04 PM MST"

	// Weekday formats with comma separator
	DisplayWeekdayComma   = "Mon, Jan 2, 2006 3:04 PM"
	DisplayWeekdayCommaAt = "Mon, Jan 2, 2006 at 3:04 PM"

	// Long formats (full weekday, full month)
	DisplayDateLong  = "Monday, January 2, 2006"
	DisplayMonthYear = "January 2006"

	// Short formats
	ShortDateTime = "Jan 2 15:04"
	ShortDate     = "Jan 2"
)
