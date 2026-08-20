package rules_test

import (
	"errors"
	"testing"

	"github.com/scriptedworld/qwark/internal/rules"
)

// ptr is how a Spec states a field, the pointer distinguishing "not stated"
// from "stated as empty".
func ptr(s string) *string { return &s }

// COVERS: FR-7.1 | positive
func TestValueMatchesTheWholeWordExactly(t *testing.T) {
	t.Parallel()

	match := rules.Value("rm")

	if !match.Matches("rm") {
		t.Error(`Value("rm") did not match "rm"`)
	}
	for _, other := range []string{"rmdir", "arm", "RM", "rm ", ""} {
		if match.Matches(other) {
			t.Errorf(`Value("rm") matched %q`, other)
		}
	}
	if match.Form() != rules.FormValue {
		t.Errorf("Form() = %q, want %q", match.Form(), rules.FormValue)
	}
}

// COVERS: FR-7.2 | positive
func TestPartialMatchesAnywhereWithin(t *testing.T) {
	t.Parallel()

	match, err := rules.Partial(".claude")
	if err != nil {
		t.Fatalf("Partial = %v", err)
	}

	for _, hit := range []string{
		"/home/ancient/.claude/settings.json",
		".claude",
		"x.claude",
	} {
		if !match.Matches(hit) {
			t.Errorf(`Partial(".claude") did not match %q`, hit)
		}
	}
	if match.Matches("/home/ancient/claude") {
		t.Error("partial matched a value not containing the text")
	}
}

// COVERS: FR-7.2 | property
func TestPartialIsTheBroadFormAndSaysSo(t *testing.T) {
	t.Parallel()

	// The predecessor blocked web.archive.org by matching the substring
	// `.archive`. Partial still does exactly that -- nothing here prevents it.
	// What changed is that the author had to name the form, so a reader of the
	// rule can see the breadth rather than deducing it from a regex.
	match, err := rules.Partial(".archive")
	if err != nil {
		t.Fatalf("Partial = %v", err)
	}

	if !match.Matches("web.archive.org") {
		t.Error("partial no longer behaves as a substring match")
	}
	if match.Form() != rules.FormPartial {
		t.Errorf("Form() = %q, want %q", match.Form(), rules.FormPartial)
	}
}

// COVERS: FR-7.5 | negative
func TestAnEmptyPartialIsRefused(t *testing.T) {
	t.Parallel()

	// Every string contains the empty string, so this clause would match every
	// command. In a deny rule that over-blocks; in an allow rule it is a hole.
	if _, err := rules.Partial(""); !errors.Is(err, rules.ErrEmpty) {
		t.Errorf("Partial(\"\") = %v, want %v", err, rules.ErrEmpty)
	}

	// An empty value is precise rather than universal, so it stands.
	if !rules.Value("").Matches("") {
		t.Error(`Value("") did not match the empty string`)
	}
}

// COVERS: FR-7.3 | property
func TestAPatternMustMatchTheWholeValue(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		expr  string
		value string
		want  bool
	}{
		{name: "exact", expr: `rm`, value: "rm", want: true},
		{name: "prefix is not enough", expr: `rm`, value: "rmdir", want: false},
		{name: "suffix is not enough", expr: `rm`, value: "arm", want: false},
		{name: "alternation", expr: `rm|rmdir`, value: "rmdir", want: true},
		{name: "alternation anchored", expr: `rm|rmdir`, value: "xrmdir", want: false},
		{name: "explicit contains", expr: `.*\.claude.*`, value: "/x/.claude/y", want: true},
		{name: "real dot", expr: `\.archive`, value: "web.archive.org", want: false},
		{name: "path prefix", expr: `/home/x/\.claude/.*`, value: "/home/x/.claude/s", want: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			match, err := rules.Pattern(c.expr)
			if err != nil {
				t.Fatalf("Pattern(%q) = %v", c.expr, err)
			}
			if got := match.Matches(c.value); got != c.want {
				t.Errorf("Pattern(%q).Matches(%q) = %v, want %v",
					c.expr, c.value, got, c.want)
			}
		})
	}
}

// COVERS: FR-7.3 | edge
func TestAnAlternationCannotEscapeTheAnchors(t *testing.T) {
	t.Parallel()

	// `\Arm|force\z` would anchor only its first branch, so any value
	// containing `force` would match. The group is what stops that.
	match, err := rules.Pattern(`rm|force`)
	if err != nil {
		t.Fatalf("Pattern = %v", err)
	}

	if match.Matches("enforcement") {
		t.Error("an unanchored branch let `enforcement` match `rm|force`")
	}
}

// COVERS: FR-7.4 | negative
func TestAPatternThatWillNotCompileIsAConfigurationError(t *testing.T) {
	t.Parallel()

	if _, err := rules.Pattern(`[unclosed`); !errors.Is(err, rules.ErrPattern) {
		t.Errorf("Pattern = %v, want %v", err, rules.ErrPattern)
	}
}

// COVERS: FR-7.6 | positive
func TestASpecStatesExactlyOneForm(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		spec rules.Spec
		want rules.Form
	}{
		{name: "value", spec: rules.Spec{Value: ptr("rm")}, want: rules.FormValue},
		{name: "partial", spec: rules.Spec{Partial: ptr(".claude")}, want: rules.FormPartial},
		{name: "pattern", spec: rules.Spec{Pattern: ptr(`rm|rmdir`)}, want: rules.FormPattern},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			match, err := c.spec.Build()
			if err != nil {
				t.Fatalf("Build() = %v", err)
			}
			if match.Form() != c.want {
				t.Errorf("Form() = %q, want %q", match.Form(), c.want)
			}
		})
	}
}

// COVERS: FR-7.6 | negative
func TestASpecStatingNoneOrSeveralIsRefused(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		spec rules.Spec
		want error
	}{
		{name: "none", spec: rules.Spec{}, want: rules.ErrNoForm},
		{
			name: "value and partial",
			spec: rules.Spec{Value: ptr("rm"), Partial: ptr("rm")},
			want: rules.ErrManyForm,
		},
		{
			name: "all three",
			spec: rules.Spec{Value: ptr("a"), Partial: ptr("b"), Pattern: ptr("c")},
			want: rules.ErrManyForm,
		},
		{name: "bad pattern", spec: rules.Spec{Pattern: ptr(`[`)}, want: rules.ErrPattern},
		{name: "empty partial", spec: rules.Spec{Partial: ptr("")}, want: rules.ErrEmpty},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			if _, err := c.spec.Build(); !errors.Is(err, c.want) {
				t.Errorf("Build() = %v, want %v", err, c.want)
			}
		})
	}
}

// COVERS: FR-7.7 | positive
func TestAMatchQuotesItselfAsWritten(t *testing.T) {
	t.Parallel()

	const expr = `/home/ancient/\.claude/.*`

	match, err := rules.Pattern(expr)
	if err != nil {
		t.Fatalf("Pattern(%q) = %v", expr, err)
	}

	// A denial quotes the rule that caused it. Showing the anchors this
	// package added would show the author something they did not write.
	if got := match.String(); got != expr {
		t.Errorf("String() = %q, want %q", got, expr)
	}
	if got := rules.Value("rm").String(); got != "rm" {
		t.Errorf("Value.String() = %q, want %q", got, "rm")
	}
}

// COVERS: FR-7.6 | edge
func TestAMatchThatWasNeverStatedTestsNothing(t *testing.T) {
	t.Parallel()

	// A zero Match can only arise by bypassing Build. It must fail closed --
	// matching nothing -- rather than matching everything.
	var unstated rules.Match

	for _, value := range []string{"", "rm", "anything at all"} {
		if unstated.Matches(value) {
			t.Errorf("an unstated match matched %q", value)
		}
	}
}
