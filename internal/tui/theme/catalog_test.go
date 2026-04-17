package theme

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuiltinThemesIncludeResolvedPalettesAndAliases(t *testing.T) {
	entries := BuiltinThemes()
	require.NotEmpty(t, entries)

	seen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		require.False(t, seen[entry.Name], entry.Name)
		seen[entry.Name] = true

		require.Equal(t, entry.Name, NormalizeName(entry.Name))
		require.NotEmpty(t, entry.Description)
		require.NotEqual(t, Palette{}, entry.Palette)
		require.Equal(t, entry.Palette, Builtin(entry.Name))

		for _, alias := range entry.Aliases {
			require.Equal(t, entry.Palette, Builtin(alias))
		}
	}
}

func TestNormalizeNameAndBuiltinFallback(t *testing.T) {
	require.Equal(t, "rose-pine", NormalizeName(" Rose_Pine "))
	require.Equal(t, "solarized-dark", NormalizeName("solarized"))
	require.Equal(t, "material", NormalizeName("material-dark"))
	require.Equal(t, "unknown", NormalizeName(" Unknown "))

	require.Equal(t, Builtin("default"), Builtin("unknown"))
	require.Equal(t, Builtin("default"), Builtin(" DEFAULT "))
}
