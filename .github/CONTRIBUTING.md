# Contributing to fabric8-wit

You are welcome to contribute to this project.  Here are few
suggestions to help you to contribute.  You can contribute to this
project by reporting issues, sending pull requests, writing
documentation etc.

If you have any questions, you can ask in the
[public mailing list](https://www.redhat.com/mailman/listinfo/almighty-public)
or
[IRC channel](http://webchat.freenode.net/?randomnick=1&channels=almighty).

## Reporting Issues

All the issues related to fabric8-wit should be reported in the
[GitHub issue tracker](https://github.com/fabric8-services/fabric8-wit/issues/new).

## Pull Requests

Your pull requests can be sent to the Github repository against the master
branch.

When a PR is ready to be merged into master, we will squash the commit history
of the PR on the fly and rebase it on top of master. The commit history of your
PR branch will be left untouched. The result is that we have one commit per PR
in our master branch.

You should treat commits on your PR that have already been reviewed with care.
That means if you want to make sure that your PR branch contains the latest
changes from master, please avoid rebasing at any cost. Instead merge master
into your branch.

We want to allow the reviewers to come back to your PR and see what has changed
since they've last reviewed it and that is only possible when you don't touch
already made commits. Remember, we will squash your commits into one single
commit on the fly before merging to master anyway. Use `git revert` for example
if you want to undo some older commit; that will cause a new commit.

As a rule of thumb, please treat your PR branch as an append-only branch without
any rewrites. That being said, if you ever encounter that you need to do `git
push -f`, then you've not followed the above guidelines. Chances are, that you
need to force push because you want to overwrite the commit history and again
you should avoid doing so. Consult your `git reflog` to understand how you can
reset your local branch that doesn't require a force push. Then you can cherry
pick your new commits and apply them on a reset branch before your push (this
time without force).

When a reviewer requests a change, please help her and make a small commit that
targets just the requested change. Please write an answer to the request that
specifies the commit ID in which you've fixed something.


## Reviewing

[This page](https://github.com/golang/go/wiki/CodeReviewComments) collects common comments made during reviews of Go code, so that a single detailed explanation can be referred to by shorthands. This is a laundry list of common mistakes, not a style guide.

## Other links on good Go programming

These links might help you when writing code or reviewing other peoples code:

* [Effective Go](https://golang.org/doc/effective_go.html)
* [Godoc: documenting Go code](https://blog.golang.org/godoc-documenting-go-code)
