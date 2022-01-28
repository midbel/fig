# fig

## todos

* variable interpolation in string
* expression evaluation: TBD
* user defined functions inside fig file: TBD

## spec

### comment

fig has two kind of comments:

* single line comment: a `#` symbol marks the rest of the line as a comment.
* multiline line comment: everything between `/*` and `*/` is a comment. nesting multiline comments is allowed but is not recommended.

### types

fig supports the following primitive types:

* string
* literal
* integer
* float
* boolean

### options

options are defined as a pair  key/value pair separated by a `=` symbol. Each option is defined on its own line. A line ends with with a `\n` or, optionally, with a `;` symbol followed by `\n`.

values can be of any of the primitive types defined above and can also be an array of these values.

```
key = value
```

an option without value is allowed. if such an option is present in a document, the parser/decoder should choose the more suitable zero-value for this field.

```
key = # empty value is allowed
```

repeating multiple times the same key is also allowed. it has as effect to create an option with the given key as identifier and an array as value with the defined values. there is no check on the type of values set in the created array.

```
key = 1
key = 2
key = true
key = foobar
# is equivalent to
key = [1, 2, true, foobar]
```

### objects

### arrays

### variables

in fig, variables can take two forms:

* local variables are identifier that reference an option defined in an object at the higher level that the object that used the variable
* environment variables are variable that are defined outside the document and pass to the parser/decoder in order to be available in the document (in its "environment")

example
```
answer = 42
hitchhiker {
    answer   = $answer # a local variable
    question = @question # an environmnet variable
}
```

### functions

### macros

macros are a simple way to modify the parsed document. The supported macros can created new options from an external file, repeated the same objects multiple times, include another fig file into the current one,...

macros in fig are a bit like function called in other programming language. Indeed, some take positional arguments (but can also be defined as keyword arguments), some not. Some macros needs an object (like the body of function), some not.

the following section describes each macros supported currently by fig as well as their arguments.

#### include

#### define

#### apply

#### extend

#### repeat

#### readfile
