# fig

fig is inspired by multiple popular and less popular configuration file format such as TOML, HCL, UCL, NGINX. But also by JSON.

### example

```
package  = "toml"
version  = @version
repo     = "https://github.com/midbel/fig"
revision = 18

dev {
  name = midbel
  email = midbel@midbel.org
}

dev projects {
  name    = toml
  repo    = "https://github.com/midbel/toml"
  version = "1.0.1"
  active  = true
}

dev projects {
  name    = hexdump
  repo    = "https://github.com/midbel/hexdump"
  version = "0.1.0"
  active  = true
}

changelog {
  date = 2021-04-03
  desc = <<DSC
  start scanning a sample fig file
  DSC
}

changelog {
  date = 2021-04-10
  desc = <<DSC
  Change how values are interpreted. Values are now intepreted as expression
  to be evaluated instead of static values.

  Add support for two kind of variables:
  * env variables provided by user when initializing the parser
  * local variables avaibles in the fig file itself
  DSC
}

changelog {
  date = 2021-04-16
  desc = "find a name for the file format"
}

changelog {
  date = 2021-04-17
  desc = "initial commit and pushing the code into its repo"
}
```
