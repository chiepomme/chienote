# chienote
An evernote-to-jekyll post synchronizer, heavily inspired by postach.io.

# Installation
`go get github.com/chiepomme/chienote`

# Usage
```
cd [your-jekyll-root]
chienote init
chienote sync
chienote convert
jekyll build
```

It is better to add `_evernote.yml` and `_cache/` to your `.gitignore`.  

# Configuration
`chienote init` initializes your configuration file, whose name is `_evernote.yml`. You need some information listed below to initialize.

- client key ( request api key at https://dev.evernote.com/doc/ )
- client secret ( request api key at https://dev.evernote.com/doc/ )
- developer token ( see https://dev.evernote.com/doc/articles/authentication.php )
- notebook name

# Formatting
In addition to evernote's text decorations, list, and todoes, you can use some markdowns.

## Heading
```
# Heading1 -> <h1>Heading1</h1>
## Heading2 -> <h2>Heading1</h2>
### Heading3 -> <h3>Heading1</h3>
#### Heading4 -> <h4>Heading1</h4>
##### Heading5 -> <h5>Heading1</h5>
```

## Code Fence
    ```go
    var someVariable int
    ```
⇩

```
{% highlight go %}
    var someVariable int
{% endhighlight %}
```

# Attachments
Attachments are just copied to `resources` directory under your jekyll root.
In posts, resources are shown by html tags.

| extension | tag |
| --- | --- |
| `*.jpg` `*.png` `*.gif` | &lt;img&gt; |
| `*.mp3` | &lt;audio&gt; |
| `*.mp4` | &lt;video&gt; |
| others | &lt;a&gt; |

# Tagging
chienote has special tags. The other tags are used as post's tags in jekyll.

| tag | meaning |
| --- | --- |
| page | rendered with `page` layout, and save under the jekyll root |
| published | make the note public |

# Custom URL
Evernote's url attribute is used for the post filename. If nothing's set, the title is used.

# Author
chiepomme  
http://chiepom.me/  
http://twitter.com/chiepomme