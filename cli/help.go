package cli

import "github.com/spolu/settle/lib/out"

// Help prints the standard help message.
func Help() {
	out.Normf("\nUsage: ")
	out.Boldf("settle <command> [<args> ...]\n")
	out.Normf("\n")
	out.Normf("Decentralized trust graph for value exchange on the Internet.\n")
	out.Normf("\n")
	out.Normf("Commands:\n")

	out.Boldf("  help [<command>]\n")
	out.Normf("    Show help for a command.\n")
	out.Valuf("    settle help trust\n")
	out.Normf("\n")

	out.Boldf("  mint <asset>\n")
	out.Normf("    Creates a new asset.\n")
	out.Valuf("    settle mint USD.2\n")
	out.Normf("\n")

	out.Boldf("  trust <user> up to <asset> <amount>\n")
	out.Normf("    Trust a user up to a certain amount of an asset.\n")
	out.Valuf("    settle trust von.neumann@ias.edu up to EUR.2 200\n")
	out.Normf("\n")

	out.Boldf("  pay <asset> <amount> to <user>\n")
	out.Normf("    Pay a user.\n")
	out.Valuf("    settle pay GBP.2 20 to von.neumann@ias.edu\n")
	out.Normf("\n")

	out.Boldf("  list <object>\n")
	out.Normf("    List balances, assets, trustlines.\n")
	out.Valuf("    settle list balances\n")
	out.Normf("\n")

	out.Boldf("  login\n")
	out.Normf("    Login to a mint (logs the current user out).\n")
	out.Valuf("    settle login\n")
	out.Normf("\n")
}
