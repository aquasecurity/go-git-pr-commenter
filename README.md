# go-git-pr-commenter

command line tool and package based for git comments

# cmd example  

GitHub: 

./commenter cmd -f file.yaml -c comment -v github --start-line 17 --end-line 20 --pr-number 9 --repo testing --owner repo_owner  

Gitlab:  

export GITLAB_TOKEN=xxxx  
export CI_PROJECT_ID=xxxx  
export CI_MERGE_REQUEST_IID=xxxx  
export CI_API_V4_URL=xxxx  
  
./commenter cmd -f file.yaml -c comment -v gitlab --start-line 18  


# Credits

Initially inspired and based on https://github.com/owenrumney/go-github-pr-commenter/ 
