---
title: Hello, world
date: 2026-05-22
---

Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod
tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim
veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea
commodo consequat.

Duis aute irure dolor in reprehenderit in voluptate velit esse cillum
dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non
proident, sunt in culpa qui officia deserunt mollit anim id est laborum.

Here's a fenced code block with a language hint, and some `inline code`
in a sentence for comparison:

```go
func greet(name string) string {
    if name == "" {
        name = "world"
    }
    return fmt.Sprintf("Hello, %s! This line is intentionally long to demonstrate how the blog handles overflow in preformatted code.", name)
}
```

And an indented block, which Goldmark also accepts:

    $ ./blog build
    built 1 posts in 3ms

Sed ut perspiciatis unde omnis iste natus error sit voluptatem
accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab
illo inventore veritatis et quasi architecto beatae vitae dicta sunt
explicabo. Nemo enim ipsam voluptatem quia voluptas sit aspernatur aut
odit aut fugit, sed quia consequuntur magni dolores eos qui ratione
voluptatem sequi nesciunt.

At vero eos et accusamus et iusto odio dignissimos ducimus qui blanditiis
praesentium voluptatum deleniti atque corrupti quos dolores et quas
molestias excepturi sint occaecati cupiditate non provident, similique
sunt in culpa qui officia deserunt mollitia animi, id est laborum et
dolorum fuga.
