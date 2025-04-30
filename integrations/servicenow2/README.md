# Cordial Servicenow v2 Integration

## Client Configuration

### Test and Transform Environment Variables

All dynamic data values are passed in to the integration as environment variables, from Geneos Actions or Effects.

The client configuration includes features to test, set and unset Servicenow fields based on the environment variables. Because these are expressed in the YAML file format it is important to use the correct hierarchical indentation and layout.

The processing is done in sections. `defaults` and profiles. Each section is processed as an ordered list of tests and actions.

### Custom Expansions

The right hand side of most entries are processed using cordial's "expandable" format with these additional custom functions:

* `${match:ENV:PATTERN}` - evaluate PATTERN as a regular expression against the contents of ENV environment variable and return "true" or "false". If ENV is an empty string (or not set) then `matchenv` returns false

* `${replace:ENV:PATTERN:TEXT}`

* `${first:ENV1:ENV2:ENV3:DEFAULT}` - return the value of the first environment variable that is set, including to an empty string. The last field is a plain string returned as a default value if none of the environment variables are set. Remember to include the last colon directly followed by the closing `}` if the default value should be an empty string

* `${snow:FIELD}` - returns to current value of the field FIELD, which will be an empty string if the field has not been set

### Actions

The following actions are available:

* `if` defaults to true if not set
* `then` starts a new sub-section which is then processed recursively
* `set` evaluates the right had side and sets the servicenow field on the left, overwriting any previous value
* `unset` removes the servicenow field from the set sent to the router
* `skip` exits the processing of the parent section

The order of actions inu a section is not important and they are processed in the following way:

#### `if`

The `if` action supports the evaluation of either a single value or an array of values, which must all be true and so acts much like `AND`-ing the values together. If the test(s) are `true` then the rest of the section is actioned, if `false` then processing of the current section stops and any further sections are then evaluated.

These kinds of layout are supported:

```yaml
defaults:
  - if: TEST
    set: ...
  - if:
    - TEST1
      TEST2
    set: ...
  - if: [ TEST1, TEST2 ]
    set: ... 
```

Note how the `set` is at the same indentation level as the `if`.

#### `then`

The `then` action introduces a subsection that is evaluated recursively.

These two `if` sections result in identical results:

```yaml
defaults:
  - if: TEST
    set: ...

  - if: TEST
    then:
      - set: ...
```

Where `then` is useful is for further tests, which are also processed in order:

```yaml
defaults:
  - if: TEST1
    then:
      - if: TEST2
          - set: 
```

#### `set`

```yaml
set:
  key: value
```

#### `unset`

#### `skip`




## Router COnfiguration
