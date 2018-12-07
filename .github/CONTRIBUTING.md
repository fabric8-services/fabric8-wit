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

#### Fork the repository
Use [GitHub’s forking feature](https://help.github.com/articles/fork-a-repo/) to get a copy of the repository into your account.

#### Clone your newly forked copy to your system
```
$ git clone git@github.com:<your-user>/fabric8-wit.git
```

#### Add a remote reference to the upstream repo
This is used for pulling in future updates, and keeping your repository up to date.
```
$ git remote add upstream git://github.com/abric8-services/fabric8-wit.git
```

#### Pull in updates from upstream
It is a good practice to regularly pull in updates from the upstream repo before starting any work. Especially if your local copy is stale.
```
$ git checkout master
$ git pull upstream master
```

#### Create a topic branch
It is a good practice to do any work in a topic branch to keep your master branch clean.
The rest of this document will assume you are using it.

```
$ git checkout -b <your-topic-branch>
```

#### Make your changes, and commit them locally
Commit your changes with a meaningful message.
```
$ git commit -m "OSIO-XXX. Your commit message."
```
If the changes you are making have a corresponding [OSIO issue](https://openshift.io/openshiftio/Openshift_io/plan)
 or [GH issue](https://github.com/fabric8-services/fabric8-wit/issues) (and they should) add the number to the commit message.

#### Sync with any upstream changes via rebase
As you are working on your branch others may have updated the upstream repository. You must synchronize with those changes by rebasing, before creating the pull request. This will apply your changes on top of any changes from upstream.
In your topic branch:
```
$ git fetch upstream
$ git rebase upstream/master
```
At this point you may run into conflicts depending on what was changed locally and upstream.
You will need to resolve any of those conflicts (try git mergetool) and rerun the rebase command.
You can abort a rebase as well with the git rebase --abort command.
> NOTE: you can also add `-i` option if you need to reorganize your commits (squash, reword).

#### Pushing your local changes to your repo
Now that you’re sync’ed with upstream, and your changes are on top of upstream master you are ready to push your local updates to your forked GitHub repository.
```
$ git push <your-fork> <your-topic-branch>
```
The push command defaults to your master branch not your current branch, so specifying your topic branch is needed to get it pushed.
> NOTE: The "-f" push option may be needed depending on the results of the rebase above. Since your PR is not yet created and under review this the force option is still possible.

#### Create the pull request
Now your updates are in your repo, and ready to share. The next step is to let the project know about them.
Create a pr against the upstream repo using the steps [described by github](https://help.github.com/articles/creating-a-pull-request-from-a-fork/)

Your pull requests can be sent to the Github repository against the master
branch.

#### PR under review
You should treat commits on your PR that have already been reviewed with care.
That means if you want to make sure that your PR branch contains the latest
changes from master, please avoid rebasing at any cost. Instead merge master
into your branch:

```
$ git fetch upstream
$ git merge upstream/master
$ git push origin HEAD:mybranch
```
We want to allow the reviewers to come back to your PR and see what has changed
since they've last reviewed it and that is only possible when you don't touch
already made commits. Remember, we will squash your commits into one single
commit on the fly before merging to master anyway. Use `git revert` for example
if you want to undo some older commit; that will cause a new commit.

> NOTE: As a rule of thumb, please treat your PR branch as an append-only branch without
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

#### Your PR is merged!
When a PR is ready to be merged into master, we will squash the commit history
of the PR on the fly and rebase it on top of master. The commit history of your
PR branch will be left untouched. The result is that we have one commit per PR
in our master branch.

## Reviewing

[This page](https://github.com/golang/go/wiki/CodeReviewComments) collects common comments made during reviews of Go code, so that a single detailed explanation can be referred to by shorthands. This is a laundry list of common mistakes, not a style guide.

## Other links on good Go programming

These links might help you when writing code or reviewing other peoples code:

* [Effective Go](https://golang.org/doc/effective_go.html)
* [Godoc: documenting Go code](https://blog.golang.org/godoc-documenting-go-code)
