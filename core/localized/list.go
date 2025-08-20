package localized

import "golang.org/x/text/language"

// List is a list of information structures for various locales.
type List[T any] []Info[T]

// Select returns the item with the best match for the given language matcher,
// as well as the selected language tag and match confidence.
func (list List[T]) Select(defaultLocale language.Tag, matcher language.Matcher) (selected T, tag language.Tag, c language.Confidence) {
	switch {
	case len(list) == 0:
		c = language.No
		return
	case matcher == nil:
		selected = list[0].Data
		tag = localeOrDefault(list[0].Locale, defaultLocale)
		c = language.Exact
		return
	case len(list) == 1:
		selected = list[0].Data
		tag, _, c = matcher.Match(localeOrDefault(list[0].Locale, defaultLocale))
		return
	}

	var available []language.Tag
	for i := range list {
		available = append(available, localeOrDefault(list[i].Locale, defaultLocale))
	}

	var index int
	tag, index, c = matcher.Match(available...)
	selected = list[index].Data

	return
}

func localeOrDefault(specifiedLocale, defaultLocale language.Tag) language.Tag {
	if specifiedLocale == language.Und {
		return defaultLocale
	}
	return specifiedLocale
}
