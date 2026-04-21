# gserver, in Go

gserver is an experimental web server adapted to serve OGDL templates along with
static content.

## To do

- Put allmost all code in library:
  - config: .conf vs ini
- Commercial or OSS main.go possible 

## Features

- The file extension of (some) files below the document root is optional
- Trailing slash and index files detection
- Markdown rendering on the fly.
- Login, Logout example functionality
- Multiple hosts

## Minimum number of dependencies

No HTTPS support (let a front-end server handle this).

- securecookies
- go-chi

## Parameter substitution in paths

While resolving a path in the document root, entries of the form _token are
used for path elements not directly found in the file system. In that case 'token'
will be available later as a variable, set to the unknown path element.

For example:

     /john/blog/1

will be sent to

    /_user/blog/_id/index.htm

if that path is present. Two variables will be available in the context:

    user=john
    id=1

## Routes



## File upload

## Templates

## Remote functions

OGDL remote functions (RPC endpoints) can be configured in .conf/config:

## Markdown processor ($MD())

TODO: describe the extensions and the \escape inline syntax for processing style
and function calls.

## Form to context

Form inputs are stored in the request's context and made directly accessible in
templates. Input names are taken as simple paths (tokens separated by dots), and
the special case where the name ends with ._ogdl is parsed as OGDL before stored
in the context.

For example, the content of the following form:

    <form>
	<input name="obj.name" />
	<input name="obj.conf._ogdl"/>

could be something like this:

    obj
      name
        "Pepe Delgado"
      conf
        ip
          192.160.1.1
        net
          255.255.255.0

## Login, Logout

There is no specific path for login or logout. Any request which does not go to
the static file handler will recognize the parameters 'Login' and 'Logout'
if present. The default login handler accepts any non-empty username and any password.

The 'redirect' parameter can be used to send the user to a specific page after
login. The default behavior is to return to the same page.

# Configuration

## Where it is

.conf directory
*.ini file

## context.ogdl

## Configurable parameters

- home directory (default .)
- default server port
-

