# go-git-pr-commenter

command line tool and package based for git comments

# cmd example  

GitHub: 

export GITHUB_TOKEN=xxxx

./commenter cmd -f file.yaml -c comment -v github --start-line 17 --end-line 20 --pr-number 9 --repo testing --owner repo_owner  

Gitlab:  

export GITLAB_TOKEN=xxxx  
export CI_PROJECT_ID=xxxx  
export CI_MERGE_REQUEST_IID=xxxx  
export CI_API_V4_URL=xxxx  
  
./commenter cmd -f file.yaml -c comment -v gitlab --start-line 18  

Azure:

export AZURE_TOKEN=xxxx  
export SYSTEM_TEAMPROJECT=xxxx
export BUILD_REPOSITORY_ID=xxxx
export SYSTEM_PULLREQUEST_PULLREQUESTID=xxxx

./commenter cmd -f /file.yaml -c best_comment -v azure --start-line 1 --end-line 1  --owner repo_organization

# Credits

Initially inspired and based on https://github.com/owenrumney/go-github-pr-commenter/ 
