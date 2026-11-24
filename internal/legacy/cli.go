package legacy

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"passenger-go/internal/vault"
)

const usage = `Usage: passenger-go migrate --from <old passenger.db> --env <old .env> [--to <new db>]

Copies every account from the old passenger-go database into a new vault.
The old files are only read, never changed.

`

// RunCLI runs the migrate subcommand and returns the process exit code.
func RunCLI(args []string, stdout, stderr io.Writer) int {
	return runCLI(args, stdout, stderr, vault.DefaultKDF)
}

func runCLI(args []string, stdout, stderr io.Writer, kdf vault.KDFParams) int {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	from := fs.String("from", "", "path to the old passenger.db (required)")
	env := fs.String("env", "", "path to the old .env holding AES_GCM_SECRET (required)")
	to := fs.String("to", "./data/passenger.db", "path of the new database; must not exist")
	fs.Usage = func() {
		fmt.Fprint(stderr, usage)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *from == "" || *env == "" || fs.NArg() > 0 {
		fs.Usage()
		return 2
	}

	res, err := Migrate(context.Background(), Options{From: *from, EnvFile: *env, To: *to, KDF: kdf, Out: stdout})
	if err != nil {
		fmt.Fprintf(stderr, "migrate: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Migrated %d accounts.\n", res.Accounts)
	fmt.Fprintln(stdout, "Your master passphrase is unchanged.")
	fmt.Fprintln(stdout, res.RecoveryKey)
	fmt.Fprintln(stdout, "Your old recovery key no longer works. Store the new one safely.")
	fmt.Fprintf(stdout, "Now delete the old database and the old .env (it holds AES_GCM_SECRET): %s, %s\n", *from, *env)
	return 0
}
