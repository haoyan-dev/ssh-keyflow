package ssh

import "strings"

func ScriptWithSuccess(system, script, marker string) string {
	if system == "windows" {
		body := ""
		if script != "" {
			body = "& {\n" + script + "\n}\n"
		}
		return body + "Write-Output '" + strings.ReplaceAll(marker, "'", "''") + "'\n"
	}
	return script + "\nprintf '%s\\n' " + Quote(marker) + "\n"
}

func OperationScript(system, operation, script string) string {
	if system == "windows" {
		return "$operation='" + operation + "'\n" + script
	}
	return ": " + operation + "\n" + script
}

func GuardedRegistration(system, probe, expected, body string) string {
	if system == "windows" {
		return "$observed=@(& {\n" + probe + "\n})\nif ($observed.Count -ne 1 -or $observed[0] -cne '" + strings.ReplaceAll(expected, "'", "''") + "') { throw 'Registration target changed' }\n" + body
	}
	return "keyflow_probe() {\n" + probe + "\n}\nobserved=$(keyflow_probe) || exit 1\n[ \"$observed\" = " + Quote(expected) + " ] || exit 1\n" + body
}

func RevocationReadBody(system, script string) string {
	if system == "windows" {
		return "$inspection=@(& {\n" + script + "\n})\nif ($inspection[-1] -ceq 'Public key present') { 'KEYFLOW_PRESENT' } elseif ($inspection[-1] -ceq 'Already absent') { 'KEYFLOW_ABSENT' } else { throw 'Unrecognized inspection' }"
	}
	return "inspection=$(\n" + script + "\n) || exit 1\ncase \"$inspection\" in 'Public key present') printf '%s\\n' KEYFLOW_PRESENT ;; 'Already absent') printf '%s\\n' KEYFLOW_ABSENT ;; *) exit 1 ;; esac\n"
}
