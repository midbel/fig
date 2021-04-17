# fig

fig is inspired by multiple popular and less popular configuration file format such as TOML, HCL, UCL, NGINX. But also by JSON.

### Example

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

# Specification

## Comments

fig supports two kind of comments:

* single line: `#`
* multiline: `/*...*/`

```
# this is a usefull comment
key = value

/* the quick brown fox
jumps over
the lazy dog.
*/
key = value
```

## Basic types

### Identifier

An indentifier should only contains ASCII letters, ASCII digits, underscore or dash and should begin with an ASCII letter.

Strings (basic and literal) and Integer are also permitted as identifier.

### String

fig supports four kind of strings:

* ident strings
* basic strings
* literal strings
* multiline strings

ident strings are strings that follows the same rules of identifier.

basic strings are surrounded by double quotes. Any character can be used inside except those that should be escaped: double quote, backslash, \r and \n. A basic string can only be written on a single line. To write a string on multiple line, you should use a multiline string described below.

literal strings are surrounded by single quotes. Like basic strings, literal strings should be written on a single line.

multiline string are similar to shell heredoc. They should begin with `<<` followed by an uppercased identifier used to delimit the beginning and end of the string.

### Boolean

Booleans are just booleans and can be written as:

* true
* false
* yes
* no
* on
* off

```
active   = false
disabled = yes
```

### Integer

to be written

### Float

to be written

### Date and Time

to be written

### Variables

fig supports two kind of variables.

* local variables are variable that are defined inside a fig file. There is however a limitation to the scope of variable. An option can only access variables that are defined at a higher level. Variables that are siblings or children are not accessible.
* env variables are variables that are defined outside of a fig file. It can be variables given to the parser or variables of the environment of the command that used fig as configuration format.

## Expression

to be written

## Option

to be written

```
key1 = 123
key2 : "a string"
```

## Object

```
section {
  int  = 123
  str  = "string"
  bool = true
}
```

to be written
