package accounts

import . "github.com/lsegal/gucumber"

func init() {
	user, pass := "", ""

	Before("@login", func() {
	})

	Given(`^I have user/pass "(.+?)" / "(.+?)"$`, func(u, p string) {
		user, pass = u, p
	})

	And(`^they log into the website with user "(.+?)" and password "(.+?)"$`, func(u, p string) {
	})

	Then(`^the user should be successfully logged in$`, func() {
		T.Skip() // pending
	})
}
