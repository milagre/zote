package alloy

import "fmt"

func validateArgs(args *Args) error {
	if args == nil {
		return fmt.Errorf("args is required")
	}
	if args.Namespace == "" {
		return fmt.Errorf("Namespace is required")
	}
	if err := args.Config.Validate(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if args.River == nil {
		return fmt.Errorf("River is required")
	}

	return nil
}
