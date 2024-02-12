# checkmail
[Golang](http://golang.org/) package for email validation.

 - Format (simple regexp, see: https://www.w3.org/TR/html5/forms.html#valid-e-mail-address and https://davidcel.is/posts/stop-validating-email-addresses-with-regex/)
 - Valid domain
 - Valid user: verify if the user and mailbox really exist

[![GoDoc](https://godoc.org/github.com/badoux/checkmail?status.png)](https://godoc.org/github.com/badoux/checkmail)

## Usage

###1# Format

    func main() {
		err := checkmail.ValidateFormat("ç$€§/az@gmail.com")
		if err != nil {
			fmt.Println(err)
		}
	}

output: `invalid format`

###2# Domain

    func main() {
		err := checkmail.ValidateHost("email@x-unkown-domain.com")
		if err != nil {
			fmt.Println(err)
		}
	}

output: `unresolvable host`

###3# User

    func main() {
		err := checkmail.ValidateHost("unknown-user-129083726@gmail.com")
		if smtpErr, ok := err.(checkmail.SmtpError); ok && err != nil {
			fmt.Printf("Code: %s, Msg: %s", smtpErr.Code(), smtpErr)
		}
	}

output: `Code: 550, Msg: 550 5.1.1 The email account that you tried to reach does not exist.`

## License

Checkmail is licensed under the [MIT License](./LICENSE).
