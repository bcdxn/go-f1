package styles

import "github.com/charmbracelet/lipgloss"

type Style struct {
	Color         Color
	Doc           lipgloss.Style
	TitleBar      lipgloss.Style
	SubtitleBar   lipgloss.Style
	ToastMsgTitle lipgloss.Style
	ToastMsgBody  lipgloss.Style
	Green         lipgloss.Style
	Purple        lipgloss.Style
	Red           lipgloss.Style
	Yellow        lipgloss.Style
}

type Color struct {
	Red               lipgloss.Color
	Yellow            lipgloss.Color
	Green             lipgloss.Color
	Purple            lipgloss.Color
	Orange            lipgloss.Color
	WetTire           lipgloss.Color
	IntermediateTire  lipgloss.Color
	HardTire          lipgloss.Color
	MediumTire        lipgloss.Color
	SoftTire          lipgloss.Color
	FiaBlue           lipgloss.Color
	Light             lipgloss.Color
	Dark              lipgloss.Color
	Subtle            lipgloss.AdaptiveColor
	PrimaryForeground lipgloss.AdaptiveColor
}

func DefaultStyles() *Style {
	red := lipgloss.Color("#CF040E")
	yellow := lipgloss.Color("#FAD105")
	green := lipgloss.Color("#17C81D")
	purple := lipgloss.Color("#DA0ED3")
	orange := lipgloss.Color("#F77C14")
	wet := lipgloss.Color("#1277EF")
	intermediate := lipgloss.Color("#2EA43F")
	hard := lipgloss.Color("#D4DFE8")
	medium := lipgloss.Color("#E4E344")
	soft := lipgloss.Color("#FA5A55")
	fiaBlue := lipgloss.Color("#0B203B")
	light := lipgloss.Color("#D1D4DD")
	dark := lipgloss.Color("#383838")
	subtle := lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	primaryForeground := lipgloss.AdaptiveColor{Light: "#383838", Dark: "#D9DCCF"}

	return &Style{
		Color: Color{
			// F1 colors
			Red:              red,
			Yellow:           yellow,
			Green:            green,
			Purple:           purple,
			Orange:           orange,
			WetTire:          wet,
			IntermediateTire: intermediate,
			HardTire:         hard,
			MediumTire:       medium,
			SoftTire:         soft,
			FiaBlue:          fiaBlue,
			// Thematic colors
			Light:             light,
			Dark:              dark,
			Subtle:            subtle,
			PrimaryForeground: primaryForeground,
		},
		Doc: lipgloss.NewStyle().Margin(1, 1),
		// header styles
		TitleBar: lipgloss.NewStyle().
			Align(lipgloss.Center).
			Bold(true).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(primaryForeground).
			Foreground(primaryForeground).
			PaddingBottom(1),
		SubtitleBar: lipgloss.NewStyle().
			Align(lipgloss.Center).
			Border(lipgloss.NormalBorder(), true, false).
			BorderForeground(primaryForeground).
			Foreground(primaryForeground),
		// toast message (i.e. race control messages) style
		ToastMsgTitle: lipgloss.NewStyle().
			AlignVertical(lipgloss.Center).
			Background(dark).
			Bold(true).
			Foreground(light).
			Padding(1, 2),
		ToastMsgBody: lipgloss.NewStyle().
			AlignVertical(lipgloss.Center).
			Background(light).
			Foreground(dark).
			Padding(1, 2),
		Green:  lipgloss.NewStyle().Foreground(green),
		Purple: lipgloss.NewStyle().Foreground(purple),
		Red:    lipgloss.NewStyle().Foreground(red),
		Yellow: lipgloss.NewStyle().Foreground(yellow),
	}
}
