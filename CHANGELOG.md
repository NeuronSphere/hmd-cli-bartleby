# Changelog

## 2026-02-25

- feat: support selecting multiple root documents via comma-separated -rd argument
- fix: html and pdf subcommands now respect the -rd flag
- fix: use default builders (html, pdf) when no bartleby manifest config present

## 2025-04-21

- fix: update hmd-cli-app version to 1.2

## 2024-10-18

- fix: bumps dep versions

## 2023-11-01

- feat: reads conf params from env vars
- feat: adds root document feature

## 2023-09-13

- fix: adds custom doc title args

## 2023-03-15

- fix: loads hmd env in update image
- feat: adds update-image command

## 2023-03-09

- feat: adds logo parameters
- feat: adds default config logo values

## 2023-03-08

- fix: adds load_hmd_env to command

## 2023-03-03

- fix: fixes missing var in puml cmd
- feat: adds configure command
- feat: adds HMD_BARTLEBY_CONFIDENTIALITY_STATEMENT

## 2023-03-01

- fix: bumps hmd-cli-app

## 2023-02-13

- fix: bumps hmd-cli-app version

## 2022-11-28

- fix: updates hmd-cli-app version
- feat: updates input path to be prj root

## 2022-09-29

- feat: bumps default img version

## 2022-07-19

- feat: updates indexes and bumps tf-bartleby version

## 2022-07-15

- feat: bumps tf-bartleby version

## 2022-03-16

- fix: corrects docker compose file

## 2022-03-11

- feat: bumps image version

## 2022-03-09

- feat: adds puml command to generate images

## 2022-03-07

- fix: creates target dir if not exists

## 2022-03-04

- feat: adds gather mode and support for NEP022

## 2022-02-11

- feat: adds support for image version override with env vars

## 2022-01-20

- fix: fixes gitignore egg-info

## 2022-01-19

- fix: remove all dependency on hmd repo home

## 2022-01-18

- feat: update repo path when no hmd repo home exists

## 2021-12-09

- feat: add autodoc flag
- fix: use the correct version to find the bartleby tf image and add docs
- test: add placeholder test
- feat: add support for multiple commands from shell input

## 2021-12-07

- feat: initial code checkin

## 2021-12-06

- feat: add python tech
- feat: generate initial repo structure
