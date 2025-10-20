#!/usr/bin/env python
# SPDX-License-Identifier: Apache-2.0
# Copyright Red Hat, Inc.

import argparse
import git
import json
import os
import shutil
import subprocess

from git import GitCommandError
from git.repo import Repo
from git.util import Actor
from pathlib import Path
from urllib.parse import urljoin


GITHUB_API = "https://api.github.com"
GITHUB_USERNAME = os.getenv('USERNAME')
GITHUB_TOKEN = os.getenv('GITHUB_PAT')

# The workflow is identified by the workflow filename or ID
WORKFLOWS_PATH = '.github/workflows'
COMMON_WORKFLOWS = ['codeql', 'scorecard', 'sonarcloud']
DEST_PROJECT_ROOT = '/tmp'


def parse_args():
    parser = argparse.ArgumentParser(description="Check repo workflows")
    parser.add_argument(
        "repo_url",
        nargs=1,
        help="GitHub repository URL (e.g., https://github.com/owner/repo)",
    )
    args = parser.parse_args()
    return args


def extract_owner_repo(repo_url):
    if repo_url.endswith(".git"):
        repo_url = repo_url[:-4]
    parts = repo_url.split("/")
    owner = parts[-2]
    repo = parts[-1]
    return owner, repo


def copy_workflow_file(workflow_filename, dest_project_path):
    """
    Copies a specified workflow file to a destination project.

    Args:
        workflow_filename (str): The name of the workflow file to copy.
        dest_project_path (str): The path to the destination project directory.
    """

    # The source workflow files will be moved to a new repo
    src_workflows_dir = os.path.join(
            os.path.abspath(__file__ + "/../../"), WORKFLOWS_PATH
    )
    dest_workflows_dir = os.path.join(dest_project_path, WORKFLOWS_PATH)
    os.makedirs(dest_workflows_dir, exist_ok=True)
    src_file_path = os.path.join(src_workflows_dir, workflow_filename)
    dest_file_path = os.path.join(dest_workflows_dir, workflow_filename)

    if os.path.exists(dest_file_path):
        print(f"{workflow_filename} already exists in {dest_project_path}.")
        return False
    else:
        try:
            shutil.copy(src_file_path, dest_file_path)
        except FileNotFoundError:
            print(f"Error: Source file {src_file_path} not found.")
        except Exception as e:
            print(f"An error occurred while copying file {src_file_path}: {e}")
    return True


def push_to_origin(repo_path, workflow, owner, repo_name):
    workflow_branch = "workflow-" + workflow
    repo = Repo(repo_path)

    print("Start to push updates to remote")
    try:
        repo.git.checkout("-b", workflow_branch)
    except GitCommandError as e:
        raise RepoException(f"Git checkout failed: {e}") from e
    repo.git.add(os.path.join(WORKFLOWS_PATH, workflow + '.yml'))

    commit_msg = "chore: add {} workflow file".format(workflow)
    committer = Actor(name=GITHUB_USERNAME, email="noreply@redhat.com")
    commit = repo.index.commit(commit_msg, committer=committer)
    remote = repo.remote()
    remote.set_url(f"https://{GITHUB_USERNAME}:{GITHUB_TOKEN}@github.com/{owner}/{repo_name}.git")
    remote.push(refspec=f"HEAD:{workflow_branch}")
    print("Done")


def check_workflows(repo_url):
    owner, repo = extract_owner_repo(repo_url)
    # Get all workflows and their states
    cmd_header = '-H "Accept: application/vnd.github+json" ' \
                 '-H "Authorization: Bearer {}" ' \
                 '-H "X-GitHub-Api-Version: 2022-11-28"'.format(GITHUB_TOKEN)
    workflows_path = '/repos/{}/{}/actions/workflows'.format(owner, repo)
    workflows_api = urljoin(GITHUB_API, workflows_path)
    cmd = 'curl -L -s {} {}'.format(cmd_header, workflows_api)
    output = subprocess.check_output(cmd, shell=True)

    workflows = json.loads(output).get('workflows')
    workflow_dict = {Path(w['path']).stem: w['state'] for w in workflows}
    print("Workflows: " + str(workflow_dict))

    for workflow in COMMON_WORKFLOWS:
        if workflow in workflow_dict:
            if workflow_dict[workflow] != 'active':
                enable_endpoint = '{}/{}/enable'.format(workflows_api, workflow + '.yml')
                cmd = 'curl -L -X PUT -s {} {}'.format(cmd_header, enable_endpoint)
                print(f"Enabling {workflow}")
                try:
                    subprocess.check_call(cmd, shell=True)
                except subprocess.CalledProcessError as e:
                    raise RuntimeError(e)
                print("Done")

        # Otherwise, add the workflow file or enable with configuration
        else:
            # For CodeQL, enable the workflow with default setup
            if workflow == 'codeql':
                # For CodeQL, configure with default setup
                codeql_endpoint = '{}/repos/{}/{}/code-scanning/default-setup'.format(
                        GITHUB_API, owner, repo)
                data = '{"state":"configured"}'
                cmd = 'curl -L -X PATCH -s {} {} -d \'{}\''.format(
                        cmd_header, codeql_endpoint, data)
                print(f"Configuring CodeQL: {cmd}")
                try:
                    subprocess.check_call(cmd, shell=True)
                except subprocess.CalledProcessError as e:
                    raise RuntimeError(e)
                print("Done")

            else:
                # Checkout the repo and copy workflow template to workflows directory
                dest_project_path = os.path.join(DEST_PROJECT_ROOT, repo)
                if not os.path.isdir(dest_project_path):
                    cmd = 'git clone --quiet {}'.format(repo_url)
                    print(f"Checking out repo {repo_url}: {cmd}")
                    try:
                        subprocess.check_call(cmd, cwd=DEST_PROJECT_ROOT, shell=True)
                    except subprocess.CalledProcessError as e:
                        raise RuntimeError(e)

                print(f"Copying {workflow} workflow file to {dest_project_path}")
                copied = copy_workflow_file(workflow + '.yml', dest_project_path)
                print("Done")
                if copied:
                    # Push the new workflow file to origin for review
                    # Here may need some customization on the copied workflow 
                    push_to_origin(dest_project_path, workflow, owner, repo)


def main():
    args = parse_args()
    repo_url = args.repo_url[0]
    check_workflows(repo_url)


if __name__ == "__main__":
    main()
