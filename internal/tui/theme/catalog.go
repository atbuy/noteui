// Package theme provides the built-in TUI theme catalog and canonical theme naming.
package theme

import "strings"

type Palette struct {
	BgColor           string
	PanelBgColor      string
	BorderColor       string
	FocusBorderColor  string
	AccentColor       string
	AccentSoftColor   string
	TextColor         string
	MutedColor        string
	SubtleColor       string
	ChipBgColor       string
	InlineCodeBgColor string
	PinnedNoteColor   string
	SyncedNoteColor   string
	UnsyncedNoteColor string
	SyncingNoteColor  string
	SharedNoteColor   string
	MarkedItemColor   string
	ErrorColor        string
	SuccessColor      string
	SelectedBgColor   string
	SelectedFgColor   string
	HighlightBgColor  string
}

func Builtin(name string) Palette {
	switch NormalizeName(name) {
	case "nord":
		return Palette{
			BgColor:          "#2E3440",
			PanelBgColor:     "#3B4252",
			BorderColor:      "#434C5E",
			FocusBorderColor: "#88C0D0",
			AccentColor:      "#88C0D0",
			AccentSoftColor:  "#97B1CC",
			TextColor:        "#ECEFF4",
			MutedColor:       "#D8DEE9",
			SubtleColor:      "#4C566A",
			ChipBgColor:      "#434C5E",
			MarkedItemColor:  "#EBCB8B",
			ErrorColor:       "#D9A2A7",
			SuccessColor:     "#A3BE8C",
			SelectedBgColor:  "#4C566A",
			SelectedFgColor:  "#ECEFF4",
			HighlightBgColor: "#3B4D6A",
		}

	case "gruvbox":
		return Palette{
			BgColor:          "#282828",
			PanelBgColor:     "#32302F",
			BorderColor:      "#504945",
			FocusBorderColor: "#FABD2F",
			AccentColor:      "#FABD2F",
			AccentSoftColor:  "#83A598",
			TextColor:        "#EBDBB2",
			MutedColor:       "#A89984",
			SubtleColor:      "#665C54",
			ChipBgColor:      "#3C3836",
			MarkedItemColor:  "#FABD2F",
			ErrorColor:       "#FC6755",
			SuccessColor:     "#B8BB26",
			SelectedBgColor:  "#504945",
			SelectedFgColor:  "#FBF1C7",
			HighlightBgColor: "#504528",
		}

	case "catppuccin", "catppuccin-mocha", "mocha":
		return Palette{
			BgColor:          "#1E1E2E",
			PanelBgColor:     "#181825",
			BorderColor:      "#313244",
			FocusBorderColor: "#CBA6F7",
			AccentColor:      "#CBA6F7",
			AccentSoftColor:  "#89B4FA",
			TextColor:        "#CDD6F4",
			MutedColor:       "#BAC2DE",
			SubtleColor:      "#6C7086",
			ChipBgColor:      "#313244",
			MarkedItemColor:  "#F9E2AF",
			ErrorColor:       "#F38BA8",
			SuccessColor:     "#A6E3A1",
			SelectedBgColor:  "#45475A",
			SelectedFgColor:  "#CDD6F4",
			HighlightBgColor: "#3D3560",
		}

	case "latte":
		return Palette{
			BgColor:          "#EFF1F5",
			PanelBgColor:     "#E6E9EF",
			BorderColor:      "#CCD0DA",
			FocusBorderColor: "#8839EF",
			AccentColor:      "#8839EF",
			AccentSoftColor:  "#1C5FE5",
			TextColor:        "#484B64",
			MutedColor:       "#65687D",
			SubtleColor:      "#BCC0CC",
			ChipBgColor:      "#CCD0DA",
			MarkedItemColor:  "#DF8E1D",
			ErrorColor:       "#CC1038",
			SuccessColor:     "#307820",
			SelectedBgColor:  "#BCC0CC",
			SelectedFgColor:  "#4B4D67",
			HighlightBgColor: "#C5BDFF",
		}

	case "solarized-light":
		return Palette{
			BgColor:          "#FDF6E3",
			PanelBgColor:     "#EEE8D5",
			BorderColor:      "#93A1A1",
			FocusBorderColor: "#B58900",
			AccentColor:      "#8F6C00",
			AccentSoftColor:  "#1E6DA5",
			TextColor:        "#3E4E53",
			MutedColor:       "#586B72",
			SubtleColor:      "#93A1A1",
			ChipBgColor:      "#E4DDC8",
			MarkedItemColor:  "#B58900",
			ErrorColor:       "#C22C29",
			SuccessColor:     "#5F6F00",
			SelectedBgColor:  "#D3CBB7",
			SelectedFgColor:  "#47595E",
			HighlightBgColor: "#C9D8E8",
		}

	case "paper":
		return Palette{
			BgColor:          "#FAFAF7",
			PanelBgColor:     "#F2F1EC",
			BorderColor:      "#D8D5CC",
			FocusBorderColor: "#4A7BD0",
			AccentColor:      "#4472C0",
			AccentSoftColor:  "#AA5526",
			TextColor:        "#2F3440",
			MutedColor:       "#656E83",
			SubtleColor:      "#C8C6BE",
			ChipBgColor:      "#E8E5DC",
			MarkedItemColor:  "#C97832",
			ErrorColor:       "#C0392B",
			SuccessColor:     "#2E7D32",
			SelectedBgColor:  "#DDD9CF",
			SelectedFgColor:  "#2F3440",
			HighlightBgColor: "#D6E4F7",
		}

	case "onedark":
		return Palette{
			BgColor:           "#282C34",
			PanelBgColor:      "#21252B",
			BorderColor:       "#3A404C",
			FocusBorderColor:  "#61AFEF",
			AccentColor:       "#61AFEF",
			AccentSoftColor:   "#C678DD",
			TextColor:         "#ABB2BF",
			MutedColor:        "#828997",
			SubtleColor:       "#3A404C",
			ChipBgColor:       "#2C313A",
			InlineCodeBgColor: "#21252B",
			MarkedItemColor:   "#E5C07B",
			ErrorColor:        "#E06C75",
			SuccessColor:      "#98C379",
			SyncedNoteColor:   "#56B6C2",
			SelectedBgColor:   "#2C3A50",
			SelectedFgColor:   "#E6EAF2",
			HighlightBgColor:  "#2B4A63",
		}

	case "kanagawa":
		return Palette{
			BgColor:          "#1F1F28",
			PanelBgColor:     "#2A2A37",
			BorderColor:      "#363646",
			FocusBorderColor: "#7E9CD8",
			AccentColor:      "#7E9CD8",
			AccentSoftColor:  "#DCA561",
			TextColor:        "#DCD7BA",
			MutedColor:       "#C8C093",
			SubtleColor:      "#54546D",
			ChipBgColor:      "#2D2D3A",
			MarkedItemColor:  "#DCA561",
			ErrorColor:       "#E56D7B",
			SuccessColor:     "#98BB6C",
			SelectedBgColor:  "#2D4F67",
			SelectedFgColor:  "#DCD7BA",
			HighlightBgColor: "#2D4F67",
		}

	case "dracula":
		return Palette{
			BgColor:          "#282A36",
			PanelBgColor:     "#21222C",
			BorderColor:      "#44475A",
			FocusBorderColor: "#FF79C6",
			AccentColor:      "#FF79C6",
			AccentSoftColor:  "#BD93F9",
			TextColor:        "#F8F8F2",
			MutedColor:       "#7A88B2",
			SubtleColor:      "#44475A",
			ChipBgColor:      "#343746",
			MarkedItemColor:  "#FFB86C",
			ErrorColor:       "#FF5555",
			SuccessColor:     "#50FA7B",
			SelectedBgColor:  "#44475A",
			SelectedFgColor:  "#F8F8F2",
			HighlightBgColor: "#44366A",
		}

	case "everforest", "everforest-dark":
		return Palette{
			BgColor:          "#2D353B",
			PanelBgColor:     "#343F44",
			BorderColor:      "#475258",
			FocusBorderColor: "#A7C080",
			AccentColor:      "#A7C080",
			AccentSoftColor:  "#DBBC7F",
			TextColor:        "#DACFB7",
			MutedColor:       "#9FAAA2",
			SubtleColor:      "#475258",
			ChipBgColor:      "#374145",
			MarkedItemColor:  "#DBBC7F",
			ErrorColor:       "#E99092",
			SuccessColor:     "#A7C080",
			SelectedBgColor:  "#425047",
			SelectedFgColor:  "#D3C6AA",
			HighlightBgColor: "#3A5247",
		}

	case "tokyo-night-storm", "tokyonight-storm", "tokyo night storm":
		return Palette{
			BgColor:          "#24283B",
			PanelBgColor:     "#1F2335",
			BorderColor:      "#414868",
			FocusBorderColor: "#7AA2F7",
			AccentColor:      "#7AA2F7",
			AccentSoftColor:  "#7DCFFF",
			TextColor:        "#C0CAF5",
			MutedColor:       "#A9B1D6",
			SubtleColor:      "#565F89",
			ChipBgColor:      "#292E42",
			MarkedItemColor:  "#E0AF68",
			ErrorColor:       "#F7768E",
			SuccessColor:     "#9ECE6A",
			SelectedBgColor:  "#364A82",
			SelectedFgColor:  "#C0CAF5",
			HighlightBgColor: "#2A3F6A",
		}

	case "github-light":
		return Palette{
			BgColor:          "#FFFFFF",
			PanelBgColor:     "#F6F8FA",
			BorderColor:      "#D0D7DE",
			FocusBorderColor: "#0550AE",
			AccentColor:      "#0550AE",
			AccentSoftColor:  "#0349A5",
			TextColor:        "#1F2328",
			MutedColor:       "#4B5563",
			SubtleColor:      "#AFB8C1",
			ChipBgColor:      "#EAEEF2",
			MarkedItemColor:  "#9A6700",
			ErrorColor:       "#A40E26",
			SuccessColor:     "#116329",
			SelectedBgColor:  "#DDF4FF",
			SelectedFgColor:  "#0E1116",
			HighlightBgColor: "#B6E3FF",
		}

	case "github-dark":
		return Palette{
			BgColor:          "#0D1117",
			PanelBgColor:     "#161B22",
			BorderColor:      "#30363D",
			FocusBorderColor: "#58A6FF",
			AccentColor:      "#58A6FF",
			AccentSoftColor:  "#79C0FF",
			TextColor:        "#C9D1D9",
			MutedColor:       "#8B949E",
			SubtleColor:      "#30363D",
			ChipBgColor:      "#21262D",
			MarkedItemColor:  "#D29922",
			ErrorColor:       "#F85149",
			SuccessColor:     "#3FB950",
			SelectedBgColor:  "#1C2B45",
			SelectedFgColor:  "#F0F6FC",
			HighlightBgColor: "#1C3A5C",
		}

	case "carbonfox":
		return Palette{
			BgColor:          "#161616",
			PanelBgColor:     "#202020",
			BorderColor:      "#3A3A3A",
			FocusBorderColor: "#78A9FF",
			AccentColor:      "#78A9FF",
			AccentSoftColor:  "#08BDBA",
			TextColor:        "#F2F4F8",
			MutedColor:       "#B0B0B0",
			SubtleColor:      "#3A3A3A",
			ChipBgColor:      "#262626",
			MarkedItemColor:  "#BE8A2F",
			ErrorColor:       "#FF8389",
			SuccessColor:     "#42BE65",
			SelectedBgColor:  "#2B2B2B",
			SelectedFgColor:  "#F2F4F8",
			HighlightBgColor: "#1E3A5C",
		}

	case "crimson":
		return Palette{
			BgColor:          "#1C1212",
			PanelBgColor:     "#251818",
			BorderColor:      "#4A2828",
			FocusBorderColor: "#D05050",
			AccentColor:      "#D05050",
			AccentSoftColor:  "#C07060",
			TextColor:        "#F0E4E4",
			MutedColor:       "#B09090",
			SubtleColor:      "#3A2424",
			ChipBgColor:      "#301E1E",
			MarkedItemColor:  "#D4A850",
			ErrorColor:       "#FF6060",
			SuccessColor:     "#6AAF6A",
			SelectedBgColor:  "#5A2A2A",
			SelectedFgColor:  "#F0E4E4",
			HighlightBgColor: "#4A1E1E",
		}

	case "dusk":
		return Palette{
			BgColor:          "#13101A",
			PanelBgColor:     "#1C1528",
			BorderColor:      "#352848",
			FocusBorderColor: "#9B6DFF",
			AccentColor:      "#9B6DFF",
			AccentSoftColor:  "#B57EF0",
			TextColor:        "#E4DDF5",
			MutedColor:       "#9A8AAF",
			SubtleColor:      "#2C2040",
			ChipBgColor:      "#231840",
			MarkedItemColor:  "#D4AF50",
			ErrorColor:       "#E05575",
			SuccessColor:     "#6AAF6A",
			SelectedBgColor:  "#4A2880",
			SelectedFgColor:  "#E4DDF5",
			HighlightBgColor: "#3A1F60",
		}

	case "rose-pine":
		return Palette{
			BgColor:          "#191724",
			PanelBgColor:     "#1F1D2E",
			BorderColor:      "#26233A",
			FocusBorderColor: "#EB6F92",
			AccentColor:      "#EB6F92",
			AccentSoftColor:  "#C4A7E7",
			TextColor:        "#E0DEF4",
			MutedColor:       "#908CAA",
			SubtleColor:      "#26233A",
			ChipBgColor:      "#2A2740",
			MarkedItemColor:  "#F6C177",
			ErrorColor:       "#EB6F92",
			SuccessColor:     "#9CCFD8",
			SelectedBgColor:  "#403D52",
			SelectedFgColor:  "#E0DEF4",
			HighlightBgColor: "#3A2D50",
		}

	case "monokai":
		return Palette{
			BgColor:          "#272822",
			PanelBgColor:     "#1E1F1C",
			BorderColor:      "#3E3D32",
			FocusBorderColor: "#A6E22E",
			AccentColor:      "#A6E22E",
			AccentSoftColor:  "#66D9EF",
			TextColor:        "#F8F8F2",
			MutedColor:       "#75715E",
			SubtleColor:      "#3E3D32",
			ChipBgColor:      "#2D2E27",
			MarkedItemColor:  "#E6DB74",
			ErrorColor:       "#F92672",
			SuccessColor:     "#A6E22E",
			SelectedBgColor:  "#49483E",
			SelectedFgColor:  "#F8F8F2",
			HighlightBgColor: "#3A4A20",
		}

	case "solarized-dark":
		return Palette{
			BgColor:          "#002B36",
			PanelBgColor:     "#073642",
			BorderColor:      "#0D4050",
			FocusBorderColor: "#268BD2",
			AccentColor:      "#268BD2",
			AccentSoftColor:  "#2AA198",
			TextColor:        "#93A1A1",
			MutedColor:       "#657B83",
			SubtleColor:      "#0D4050",
			ChipBgColor:      "#0D3E4A",
			MarkedItemColor:  "#B58900",
			ErrorColor:       "#DC322F",
			SuccessColor:     "#859900",
			SelectedBgColor:  "#0D4A5A",
			SelectedFgColor:  "#EEE8D5",
			HighlightBgColor: "#083C54",
		}

	case "ayu-dark":
		return Palette{
			BgColor:          "#0D1017",
			PanelBgColor:     "#131721",
			BorderColor:      "#1D2433",
			FocusBorderColor: "#E6B450",
			AccentColor:      "#E6B450",
			AccentSoftColor:  "#39BAE6",
			TextColor:        "#BFBDB6",
			MutedColor:       "#626A73",
			SubtleColor:      "#1D2433",
			ChipBgColor:      "#161D2A",
			MarkedItemColor:  "#E6B450",
			ErrorColor:       "#F26D78",
			SuccessColor:     "#7FD962",
			SelectedBgColor:  "#1E2A40",
			SelectedFgColor:  "#E6E1CF",
			HighlightBgColor: "#1E2E40",
		}

	case "material":
		return Palette{
			BgColor:          "#212121",
			PanelBgColor:     "#292929",
			BorderColor:      "#3C3C3C",
			FocusBorderColor: "#82AAFF",
			AccentColor:      "#82AAFF",
			AccentSoftColor:  "#89DDFF",
			TextColor:        "#EEFFFF",
			MutedColor:       "#717CB4",
			SubtleColor:      "#3C3C3C",
			ChipBgColor:      "#303030",
			MarkedItemColor:  "#FFCB6B",
			ErrorColor:       "#F07178",
			SuccessColor:     "#C3E88D",
			SelectedBgColor:  "#2D3A5A",
			SelectedFgColor:  "#EEFFFF",
			HighlightBgColor: "#263250",
		}

	case "monochrome":
		return Palette{
			BgColor:           "#000000",
			PanelBgColor:      "#0d0d0d",
			BorderColor:       "#2a2a2a",
			FocusBorderColor:  "#666666",
			AccentColor:       "#aaaaaa",
			AccentSoftColor:   "#777777",
			TextColor:         "#e8e8e8",
			MutedColor:        "#666666",
			SubtleColor:       "#333333",
			ChipBgColor:       "#1a1a1a",
			InlineCodeBgColor: "#111111",
			PinnedNoteColor:   "#ffffff",
			SyncedNoteColor:   "#aaaaaa",
			UnsyncedNoteColor: "#606060",
			SyncingNoteColor:  "#ffffff",
			SharedNoteColor:   "#d0d0d0",
			MarkedItemColor:   "#ffffff",
			ErrorColor:        "#ff5555",
			SuccessColor:      "#bbbbbb",
			SelectedBgColor:   "#222222",
			SelectedFgColor:   "#ffffff",
			HighlightBgColor:  "#1c1c1c",
		}

	case "nightfox":
		return Palette{
			BgColor:          "#192330",
			PanelBgColor:     "#212E3F",
			BorderColor:      "#29394F",
			FocusBorderColor: "#719CD6",
			AccentColor:      "#719CD6",
			AccentSoftColor:  "#9D79D6",
			TextColor:        "#CDCECF",
			MutedColor:       "#738091",
			SubtleColor:      "#29394F",
			ChipBgColor:      "#253446",
			MarkedItemColor:  "#DBC074",
			ErrorColor:       "#C94F6D",
			SuccessColor:     "#81B29A",
			SelectedBgColor:  "#223549",
			SelectedFgColor:  "#CDCECF",
			HighlightBgColor: "#1E3A50",
		}

	default:
		return Palette{
			BgColor:          "#1E1E1E",
			PanelBgColor:     "#2A2A2A",
			BorderColor:      "#5F5F5F",
			FocusBorderColor: "#8866CC",
			AccentColor:      "#8866CC",
			AccentSoftColor:  "#9E7CC0",
			TextColor:        "#E5E5E5",
			MutedColor:       "#A8A8A8",
			SubtleColor:      "#444444",
			ChipBgColor:      "#3A3A3A",
			MarkedItemColor:  "#D7AF5F",
			ErrorColor:       "#D75F5F",
			SuccessColor:     "#87AF87",
			SelectedBgColor:  "#3D2272",
			SelectedFgColor:  "#FFFFFF",
			HighlightBgColor: "#2D1850",
		}
	}
}

// BuiltinThemeEntry describes a built-in theme available for use in config.toml.
type BuiltinThemeEntry struct {
	// Name is the value to set as theme.name in config.toml.
	Name string
	// Aliases lists alternate names accepted by the config parser.
	Aliases []string
	// Description is a short human-readable summary of the theme's look and feel.
	Description string
	// Palette is the resolved color palette for the theme.
	Palette Palette
}

// BuiltinThemes returns the list of all built-in themes with their palettes and descriptions.
func BuiltinThemes() []BuiltinThemeEntry {
	entries := []struct {
		name    string
		aliases []string
		desc    string
	}{
		{"default", nil, "Dark theme with deep purple accents"},
		{"nord", nil, "Arctic dark theme with cool blue and slate tones"},
		{"gruvbox", nil, "Retro dark theme with warm amber and earthy browns"},
		{"catppuccin", []string{"catppuccin-mocha", "mocha"}, "Soothing dark theme with soft pastel colors (Catppuccin Mocha)"},
		{"latte", []string{"catppuccin-latte"}, "Warm light theme with gentle pastel tones (Catppuccin Latte)"},
		{"solarized-light", nil, "Light theme using the iconic Solarized color palette"},
		{"paper", nil, "Minimal light theme inspired by ink on paper"},
		{"onedark", nil, "Dark theme inspired by Atom's One Dark colorscheme"},
		{"kanagawa", nil, "Dark theme with warm earth tones evoking Japanese woodblock art"},
		{"dracula", nil, "Dark theme with vibrant neon pink and purple accents"},
		{"everforest", []string{"everforest-dark"}, "Comfortable dark theme with natural muted green tones"},
		{"tokyo-night-storm", []string{"tokyonight-storm"}, "Dark theme evoking neon city lights on a rainy Tokyo night"},
		{"github-light", nil, "Light theme faithful to GitHub's interface colors"},
		{"github-dark", nil, "Dark theme faithful to GitHub's interface colors"},
		{"carbonfox", nil, "Sleek dark theme with IBM Carbon design aesthetics"},
		{"crimson", nil, "Deep dark theme with rich crimson and wine-red accents"},
		{"dusk", nil, "Dark theme with twilight purples evoking the last light of day"},
		{"rose-pine", []string{"rosepine", "rose_pine"}, "Dark theme with dusty rose and muted purple tones (Rosé Pine)"},
		{"monokai", nil, "Classic dark theme with vivid green, pink, and cyan accents"},
		{"solarized-dark", []string{"solarized"}, "Dark counterpart to Solarized using the same precise palette"},
		{"ayu-dark", []string{"ayu"}, "Minimal dark theme with a warm golden accent (Ayu Dark)"},
		{"material", []string{"material-dark"}, "Dark theme inspired by Google's Material Design color system"},
		{"nightfox", nil, "Soft navy dark theme with blue and purple tones (Nightfox)"},
		{"monochrome", nil, "Pure black and white theme with no color accents"},
	}
	result := make([]BuiltinThemeEntry, len(entries))
	for i, e := range entries {
		result[i] = BuiltinThemeEntry{
			Name:        e.name,
			Aliases:     e.aliases,
			Description: e.desc,
			Palette:     Builtin(e.name),
		}
	}
	return result
}

// NormalizeName returns the canonical theme name for any accepted alias.
func NormalizeName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "catppuccin-mocha", "mocha":
		return "catppuccin"
	case "catppuccin-latte":
		return "latte"
	case "rosepine", "rose_pine":
		return "rose-pine"
	case "ayu":
		return "ayu-dark"
	case "solarized":
		return "solarized-dark"
	case "material-dark":
		return "material"
	default:
		return strings.ToLower(strings.TrimSpace(name))
	}
}
