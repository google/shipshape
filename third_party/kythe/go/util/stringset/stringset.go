// Package stringset contains a simple set implementation
package stringset

// A Set is a compact set of strings
type Set map[string]struct{}

// New returns a new Set containing the given strings
func New(init ...string) Set {
	set := make(Set)
	for _, str := range init {
		set.Add(str)
	}
	return set
}

// Slice returns a slice of the strings contained with the set
func (s Set) Slice() []string {
	res := make([]string, 0, len(s))
	for str := range s {
		res = append(res, str)
	}
	return res
}

// Add will insert str into the set, if it does not already exist
func (s Set) Add(str string) {
	s[str] = struct{}{}
}

// Remove will remove str into the set, if it exists
func (s Set) Remove(str string) {
	delete(s, str)
}

// Contains returns true if str is within the set
func (s Set) Contains(str string) bool {
	_, ok := s[str]
	return ok
}
