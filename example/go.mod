module example

go 1.13

require (
	github.com/charmbracelet/bubbles v0.13.0
	github.com/charmbracelet/bubbletea v0.22.2-0.20220830200705-989d49f3e69f
	github.com/charmbracelet/lipgloss v0.5.0
	github.com/knz/catwalk v0.0.0-20220831193209-b17ece3d9ab2
	github.com/muesli/termenv v0.11.1-0.20220212125758-44cd13922739
)

replace github.com/knz/catwalk => ./..
