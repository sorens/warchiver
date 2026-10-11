package api

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Sluggable struct {
	name string
	slug string
}

func NewSlugFromName(s string) *Sluggable {
	// note: lossy
	return &Sluggable{name: s, slug: strings.ToLower(strings.ReplaceAll(s, " ", "-"))}
}

func NewSlugFromSlug(s string) *Sluggable {
	tmp := []string{}
	caser := cases.Title(language.English)
	for _, s1 := range strings.Split(s, "-") {
		tmp = append(tmp, caser.String(s1))
	}
	// note: best guess
	return &Sluggable{name: strings.Join(tmp, " "), slug: s}
}

func (s *Sluggable) Slug() string {
	return s.slug
}

func (s *Sluggable) Name() string {
	return s.name
}

func (s *Sluggable) String() string {
	if s.name != "" {
		return s.name
	}

	return s.slug
}
