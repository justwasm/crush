package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/crush/internal/ui/logo"
	"github.com/spf13/cobra"
)

var letterformsCmd = &cobra.Command{
	Use:   "letterforms",
	Short: "Print all available letterforms used in the Crush logo",
	Long:  `Render and display every letterform used in the Crush logotype, both stretched and unstretched. Useful for previewing or debugging letter designs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		stretch, _ := cmd.Flags().GetBool("stretch")

		letters := []string{"A", "C", "E", "EAlt", "G", "H", "M", "O", "P", "R", "SAlt", "T", "U", "Y", "YAlt"}
		letterforms := map[string]func(bool) string{
			"A":     logo.LetterA,
			"C":     logo.LetterC,
			"E":     logo.LetterE,
			"EAlt":  logo.LetterEAlt,
			"G":     logo.LetterG,
			"H":     logo.LetterH,
			"M":     logo.LetterM,
			"O":     logo.LetterO,
			"P":     logo.LetterP,
			"R":     logo.LetterR,
			"SAlt":  logo.LetterSAlt,
			"T":     logo.LetterT,
			"U":     logo.LetterU,
			"Y":     logo.LetterY,
			"YAlt":  logo.LetterYAlt,
		}

		for _, name := range letters {
			fmt.Fprintf(os.Stdout, "--- %s ---\n%s\n\n", name, letterforms[name](stretch))
		}

		return nil
	},
}

func init() {
	letterformsCmd.Flags().BoolP("stretch", "s", false, "Render letterforms in their stretched form")
}