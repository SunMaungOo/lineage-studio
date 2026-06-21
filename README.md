# Repo Configuration

1. Set up the repo with initer program (assuming it is generate into repo/my_repo)
2. Configure the repo that initer program generate with `` git init , git config user.name , git config user.email , git add , git commit``
3. Create the directory ``remote/my_repo``
4. Copy content of ``repo/my_repo`` into ``remote/my_repo`` (including .git)
5. In ``repo/my_repo`` , set up the remote ``git remote add orgin remote/my_repo`` (the remote name must be origin)
6. In ``remote/my_repo`` , configure to be bare repo by ``git config --local core.bare true``