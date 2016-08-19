// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package main

import (
	"encoding/json"
	"strings"
	"testing"

	"mig.ninja/mig"
)

var keyValidUser1 = `
-----BEGIN PGP PUBLIC KEY BLOCK-----
Version: GnuPG v1

mQENBFe2DpMBCADW6v/EsrimvQJhMDhoUEuf21Gc9j6c69fKW3Fa5Hvt/Tp1pb2v
IW7HeL/qVadZ49xfxMCoyJ+Ygf0tzcWwsgOFeVLRU2Okl5knZx9Uo7vP5EvhiRlM
GtNCYYB5dDQnYcz+LrkRFOsfKr7v66acjOSiuxfbKmg+iLfJRc6zrTN6Ql5g7fkJ
SahnbbAKo1qNOAyZEwBq/zHuTkZiBWcigQuFtc1dzLnk3cfDQ2qWf5p1z3jlxxP6
Qz5EIXaNf187BtrwEC0tqlZxfNYvx1Yka4D4MkG/YT3Vfmx89jT2EswkDZUY4jpl
9r1v/M0Amd+CXb7+k2ZSyRKo4s+LwK/XhB3pABEBAAG0CnZhbGlkdXNlcjGJATgE
EwECACIFAle2DpMCGwMGCwkIBwMCBhUIAgkKCwQWAgMBAh4BAheAAAoJENodUeWs
y21/Z7gIAJsBFPk5j+Y5kflCNIj3a7qgZOVz0XEy9GZw5WafwDcAIbYw1UkPWnxd
blsly05R3QVHJdzsSLkGS3NbljPvi07+A4ld2n0QqySX4rIzhzcKKYwYLnmfMb3o
MRgAJijHF3Ddzfc5KBQmphgEScKK34tGyffrCORUPBmXG53skr9zlBnS7VAuLKYi
HJ72vlpUJ2n7YNJGQ07wvPgEuGSYXiQrVuzeZ/suZwluFFLj6QCXHFqvWtD1cW9G
+FmNmYCCJbZPRddsrW588zFBcqpxvrN3I0ODGLUe2XMjGdyY8uoOEYfBpGJiuAp8
Go+r+YmG684YZ2plIiY3qQ9d/AenZay5AQ0EV7YOkwEIANh+A14PkCDTSXF185Bi
pT/quv6lEgY+98UtGOGFGZa7YsjEe9hJEZPr9uN0Wf5AqIcBB+/AUuB2+24YIJxq
Qx0tzWJLp1C0EBcSLlQejzUpCspMsZMMBLies4nHmDSP025+gOFyHYj0gnLcYSzv
7bY1b6PT7RKy28L9p7MeAO/mYlCIrc9qe7n9XJ0/38dvKldxWVE0jW2FChb/9D+X
oGAMpkTut1Y70i3v1QUYjmUkFf1e+PKwO8QCcPC0o3w0mTJF/QKgT6dQzw9C3eMg
QJV/CGJGeWN59Yw/anPQX4xKvylEFCRYObUsr6fx+7/xWtcBzfJObavdKIcP7TI7
nHsAEQEAAYkBHwQYAQIACQUCV7YOkwIbDAAKCRDaHVHlrMttf3hsB/4iN6qOTxCO
fCrDSJ1jrbd0tUgDM0nfVUcphI9Q88UZugJNyYL7c6KVyNXC3mQEclrokZYyfCtm
hmRGGNcqpgpd4U5y+nuqgWdGOiCws/c0Gh2y9el08A2ez7cAxcvShyN9BTzMTKae
cfJgAQ3IpPeQlxZR1SArzX9yKZtZUSlgxEb7ZULPX2aK2t47G7dkQUQA2m0emdvv
XBHchjWUf74QUkg81j9tKpzqbfp49cT++edEAj70hh7ET7wr43brDraWryTBztA5
V5ke3OHTM8+mvPGBQGePjRU36zcThpu93e5gKNtR/EWObc/kExUg1lEuvl+0t31m
FuA4DU63IF1v
=jpvG
-----END PGP PUBLIC KEY BLOCK-----`

var keyValidUser2 = `
-----BEGIN PGP PUBLIC KEY BLOCK-----
Version: GnuPG v1

mQENBFe2DzMBCACvy6/BPkM2uhMd55atdUuuFYNYG/5WiUGwHC2+rzaQxggA1/eF
/eolt5tKH1cg6WfKUpzQY1ZWfegnSe4VlK7283itQ8tvCc1iJtVUfJN6uFhJLzVZ
AiSHqBN9mlep0k+8RJfH5U3noLHA9rhEGrTfJd6VfRrnVg0nf3ucsiBYZ+G54DwT
4OrvGcHBLoZsbcKfaCR68WmOLjy3eO5u0OHe0YwBqmPW/WTMzHJhPe7Dwmv+yqA+
NT8JBDAYPF+WBrvU1NbKz1aUli3Eoqeqm4Zl6BG0DpbtPvMQ8BMV7W3bKD8y0Buv
hT8Qfjp4wCkBeYlXkSJbvkBcN/063Q9l2QLhABEBAAG0CnZhbGlkdXNlcjKJATgE
EwECACIFAle2DzMCGwMGCwkIBwMCBhUIAgkKCwQWAgMBAh4BAheAAAoJELnv90Az
iQ/X6yYH/35Wa1rMzxwPpYBYAGX+/jkoK4aLRkdhDpS+00ScK5T1CtrUyDDnIKzN
Ar/VYBtelsBdzFYUyUz6Sr0fYp/fX2gwS+Lay+thXlH3TieUbFk4V8P/fXANzMkt
+JcFmxhDjObUAJSq6Uptnkwa5P9ChM4iHOwMmrCZLCYHUFCLVGsY3wPessX9RjRK
clQoyOhUpR+5dOup++w95YHZqrAWdBiolIX3H1BLg5B3B7d+K+8WkSx3oPqmi8tM
Cbg5o96LCZ7y1luHmhtg671+m1P50vDpodhtZco/8URN83O4OPICAucMwfVUjOh1
eqKYrZQFxMQ2wGQRu9yFA6Dgvd49CF25AQ0EV7YPMwEIAKznZWnQnj2EN616Tk4D
H9mt+VHcZXvgdbX5WjNwJk6Mkhj6nP0Hb9OZPqAW5yxrt3ewOaME3ld/inRt4EEb
zzAVNQWHcT6t3s7HWH5AdDhgx3Yl9Os4amJnynebM+8tqlpLaYkAiT4EfCUkhzYN
hleW6UK+NLLMvI4yMKmE75GVjHSdJj+EnmgcENsmeYpNDYo2Ylh5GG3hmqxc0ZGg
FlPNYCIeoxzQSbNFBXWLn6U0I+2RNmC+Kpx/DDF1Tvgkpsfd5d2QQzEC9etGyV+r
Bdg3k8hSyaGv5u3rjibqUdkC2lJcsIhsa/tM2TaOFUNMWrA4ADdhDSFKfwb8LQDf
ED8AEQEAAYkBHwQYAQIACQUCV7YPMwIbDAAKCRC57/dAM4kP11liB/9fvOtLJ8De
/Wo45CFfotVJo39P/X3/rWMvmTWBJ7IqdYSzFJoUdJYzVCoLfeAuh1n44rvRFGu0
M2u0Q7RRk1eCYpV4BFyQPfmhVJpC19Y8C0HJ38G9tN4SVH5wJqjDJg7RIEBTodyg
T7XTGYlTW0h0jsbA0Ho4zt6wDnPh70PlDgLd++LEY9yrDxPlFdljabAH6bzVY8rF
1PqDms869uzFw6LvZ/F35e6PdhP2iI8elwuhYtXPmkstKs3leN3r9bvKecazs2Bo
C7X+Ca4N4gnCaVHc3lrrJfnQ0SEcrJZ4Sg9IY2HU0wvzPmWIdc2hXoUkK3KQClmo
b6QoxMi4WWAE
=jsXH
-----END PGP PUBLIC KEY BLOCK-----`

var keyInvalidUser1 = `
-----BEGIN PGP PUBLIC KEY BLOCK-----
Version: GnuPG v1

mQENBFe2D0IBCADOYO1ibNKhV21EDoK/WC3pNtCA5AjMUATyQ1A227AyFVfuRJlj
vsd3LEY+UsnbDyLk1parrL1/MjVnwyZOQsTDrOvWSmN4griXxAB+v+FOK65XxuMN
y9a1IJWxVFfoem/5YO7AyifFS2Ayl2VJkKf+LFq9jWTOk7ZbYn2z8dJittR/1glm
izb5RH5jDlctfeghwrDfZmNBw3bieCzbiX5ZQugqioSeKyvB75OSKb5ArRwynjjH
6DEplC+NuQluiBKfnZ5YQKxn2JnEzcSrY/42XRX5HPCA4cmGHuDk1dmbtTRxo5Lr
yNLLYtwWVs/4rVrz3xddW5v+I+xpjpQnvJ3tABEBAAG0DGludmFsaWR1c2VyMYkB
OAQTAQIAIgUCV7YPQgIbAwYLCQgHAwIGFQgCCQoLBBYCAwECHgECF4AACgkQKJlT
VKzjwsExKggAiICKBlpPnnN8h+scS0Xs7kJlg6734UnDsNH9MLQS+wop+G9KQJEL
pt5IzClbwi8p9+2MLEbKcj241VV1W/otP4HHPlABKjoDblal7RLsfElj7i2nUDbJ
ob1N7G/hOJPSnGSadX6D6Tea2sctYJWWIKjvkcecx/Rwf7U1+hI0V0bQLR+v7h+c
63/GAAqru3IZCMhuSgiC+k+V54TGKjqmbIMAJ1vl2+MlpTkj+Luh+647KjlTJHjq
2EKLHSxkH2rmIeaB1SSZlG8p+Wn3QIUlV0rlhLI89vNcVnfVCf14DGSQ7xNaj2Yd
hWVtmZeKT8dleKl1KMedznNXYyqVQUI1obkBDQRXtg9CAQgAuAvno+yjZpbV77VL
cq0Vz8/Wnx2WwQsosP2K3TO5jfUvaWoaalKWXq1HDiwPfsn5SUArJYNyqscl7loo
I44Q+7zmlbIVlQ269W84e5PerjNGb+JXZQLC1jiocVnAM2DQ8FilDhsqLrCfQ4zz
FpN6B+eiCGwokcMNkGQkkhyBv/4xFNVVjj/+NyWLDcDChPv5V756h3wpW/zmIEC3
Lq4hchV4WoMB7aMDqO9u5IeHyOsEJBlOz39GYfwyTP8UKo7Xs76WbLQRQp4KGf+u
3PSpLHJzWrdUah+KMXMhIS8GRHgaMIPfS7SVvMKEQ147/xsbj7HTLNJ7SQW60+se
eb5cawARAQABiQEfBBgBAgAJBQJXtg9CAhsMAAoJECiZU1Ss48LBU9QIAIC4Y1p5
rrA9TL1Hf1FEW/jvGtlfz1Jei3juAsQEcBS+FS+GOSY8XEhSJ0fq0hqMWuFQgSEm
/mxTIBRsRjV8EboQ0HKTyTFtZO1kN2R9mVa9U++T1SkfPRQTVO/g4fa3SnRw4L0I
9q/3OB+gxtaimV05nJFA4dmEccmsiDCXxQyR89KPNEJEb7ZT56NWYWmoerUIEVpc
ADOLIkgFZ529i+s0BnPqn7hCMy1ujcDtzWWFHTIRS292OLqfbQ6AeX0vE5ZAZVzr
QZXJ2IeDh4ZWZ2+DC1+mOhRZ1SpuEId5MgIFtcLrvhFDBljX2ItZg4W//VGZALmL
jVWqBem9ujvb1/c=
=ReHd
-----END PGP PUBLIC KEY BLOCK-----`

var manInsufficientSigs = `
{
  "loader_name": "",
  "entries": [
    {
      "name": "mig-loader",
      "sha256": "355b34f99acaeb1ab18a7e08b7f19d54b7915fb234777bee828ed5e34b30c9de"
    },
    {
      "name": "configuration",
      "sha256": "6dcf3bc90eb5c76528edcb1f71bed7dccb886463a5c0c19b33bc1e8d25ce82d8"
    },
    {
      "name": "mig-agent",
      "sha256": "ce7e7588073021d590ede399dcf31f9148da969e333085a3e58ef78f9d8c31f2"
    },
    {
      "name": "agentcert",
      "sha256": "017525f2f851311e9b0e26a139252c13b186a6507206cbd0dcc1ca35258b9566"
    },
    {
      "name": "cacert",
      "sha256": "215394a591db4dbf2bbbb17a4d45b5bc6d335d15a7d2c42876d4b27f8269bda9"
    },
    {
      "name": "agentkey",
      "sha256": "88df8f032916dfa0ae6c4778fd2aa2084c1aac017aab70f7d4bc6f4327c5c24c"
    }
  ],
  "signatures": [
    "wsBcBAABCAAQBQJXthNjCRDaHVHlrMttfwAAxaoIAFq4OSBL0kGGLqVhobmHXqqxLvEbOhdpcT5PsYM
    hi+CMzE429mPatqB2PukZrTjg9z2dJgCOreGMk3PdeEG3HttfHZoXKc73jheZiwewSXGkopWxlBNs35t
    +FqurpGggJTJ9M8MtK7orCxFE/ei3AYRu6vELhUz+0A5EpI2Fwuo5stiGQGxpNQG4QhZIphbZng7PThb
    9Y2f1WkwUoyiDRmzjnrDCt9XYXbrywUSfWawWMmBW/Qmq58IQNmxWGK3mZq1/oQJdextS24J1LJeuCR4
    1+EcqsLS9P7ujFyQD3fbKOPvk8krzFdVHpRafevXwaORc6j2hO9vVV/kgzZ2Ym8A==hdtT"
  ]
}`

var manCorrectSigs = `
{
  "loader_name": "",
  "entries": [
    {
      "name": "mig-loader",
      "sha256": "355b34f99acaeb1ab18a7e08b7f19d54b7915fb234777bee828ed5e34b30c9de"
    },
    {
      "name": "configuration",
      "sha256": "6dcf3bc90eb5c76528edcb1f71bed7dccb886463a5c0c19b33bc1e8d25ce82d8"
    },
    {
      "name": "mig-agent",
      "sha256": "ce7e7588073021d590ede399dcf31f9148da969e333085a3e58ef78f9d8c31f2"
    },
    {
      "name": "agentcert",
      "sha256": "017525f2f851311e9b0e26a139252c13b186a6507206cbd0dcc1ca35258b9566"
    },
    {
      "name": "cacert",
      "sha256": "215394a591db4dbf2bbbb17a4d45b5bc6d335d15a7d2c42876d4b27f8269bda9"
    },
    {
      "name": "agentkey",
      "sha256": "88df8f032916dfa0ae6c4778fd2aa2084c1aac017aab70f7d4bc6f4327c5c24c"
    }
  ],
  "signatures": [
    "wsBcBAABCAAQBQJXthNjCRDaHVHlrMttfwAAxaoIAFq4OSBL0kGGLqVhobmHXqqxLvEbOhdpcT5Ps
    YMhi+CMzE429mPatqB2PukZrTjg9z2dJgCOreGMk3PdeEG3HttfHZoXKc73jheZiwewSXGkopWxlBN
    s35t+FqurpGggJTJ9M8MtK7orCxFE/ei3AYRu6vELhUz+0A5EpI2Fwuo5stiGQGxpNQG4QhZIphbZn
    g7PThb9Y2f1WkwUoyiDRmzjnrDCt9XYXbrywUSfWawWMmBW/Qmq58IQNmxWGK3mZq1/oQJdextS24J
    1LJeuCR41+EcqsLS9P7ujFyQD3fbKOPvk8krzFdVHpRafevXwaORc6j2hO9vVV/kgzZ2Ym8A==hdtT",
    "wsBcBAABCAAQBQJXtiG4CRC57/dAM4kP1wAAxpIIAAqsrcnBWgci5PLkqjdEorUX6J4Dm+G6VXc6f
    U8mKd+jHNWxYwMj42eRY+byEHCNkWY0gyCm9epKNltZ5fLv9LmcQZrqqy7mk37MbZKubiSrMSvFyfO
    mrFkKsYS+JdTzXukkmWc/nU5WUtvfe3cnvGhs8FFBuf3v9Xo/d0m0yqPKszNYkb12ZiXFynu1MhdB8
    vQp4u0JrEBhyR+MC2wLhYv/zfzIMLA4CJZaleUfgHyBSAc6FeJyuR7E0e1g6ORHcDJmt8fpJpH7heU
    Kr2/aLMtK8fnqijIt5UDEcxzWEVjbYY5/elofrgEAgaf3V/e4ILHSBve9tc7LN9ZwsHYvsyE==HYI1"
  ]
}`

var manOneInvalidSigner = `
{
  "loader_name": "",
  "entries": [
    {
      "name": "mig-loader",
      "sha256": "355b34f99acaeb1ab18a7e08b7f19d54b7915fb234777bee828ed5e34b30c9de"
    },
    {
      "name": "configuration",
      "sha256": "6dcf3bc90eb5c76528edcb1f71bed7dccb886463a5c0c19b33bc1e8d25ce82d8"
    },
    {
      "name": "mig-agent",
      "sha256": "ce7e7588073021d590ede399dcf31f9148da969e333085a3e58ef78f9d8c31f2"
    },
    {
      "name": "agentcert",
      "sha256": "017525f2f851311e9b0e26a139252c13b186a6507206cbd0dcc1ca35258b9566"
    },
    {
      "name": "cacert",
      "sha256": "215394a591db4dbf2bbbb17a4d45b5bc6d335d15a7d2c42876d4b27f8269bda9"
    },
    {
      "name": "agentkey",
      "sha256": "88df8f032916dfa0ae6c4778fd2aa2084c1aac017aab70f7d4bc6f4327c5c24c"
    }
  ],
  "signatures": [
    "wsBcBAABCAAQBQJXtiKTCRC57/dAM4kP1wAAsPcIAH2vraXci+ZxkOtzjdBraZjx+CK3a/GB4GIid
    oteZ+nwzBI98gg8byWl2VsOgR+8DrwMxmtbVykzu5eg9KvdZlOFMAm79MD23TfJL+iFu/2uwzK8IWk
    TD7rJ6PuFQzZu/KhqnLvM+96xsUke5BASpDbnwSz+JYZH/PgtY7qYuGvS2DnKjEDXFczIjnDWAaSvh
    1/kwOhzAiKtRnlO/PmE4vzI3/mGX2pdX3XFQY1LMW2oE6fOgbz40+VXwBh+GRYQmb0nDaC22i10iLB
    e4V0AGGQiKa5SacEcqBgSg1TYd1T9JgBLD8dOvwUGPcTdct9Cm23LTjXJyl4fMLpgvQVuCcY==mD86",
    "wsBcBAABCAAQBQJXtiKmCRAomVNUrOPCwQAAt9oIABm0y5zOg/kM6/eWOxA245W6Tqs51ERP5pbXZ
    cCni6ozQZPQw0DI3xa8RyupYlyRi4Ud6GdXNaakYLesHXQPu4ICZxLejtEo9gYFTczl04qfLr+vAB1
    nMU6XkLw3T3GZPYdGbKcfZ/+P8Yo/DywayzkYiYBQsK50xunGwADfHUcoNNjJij/oPm5lKmEF6fi2U
    HwX2ugHGcjGtanytPyNJVaWwtf3zOHPK9xkY9JzjsmVqD+lAjRMe63WzpZxp8MIrwWT7G++a54ohD6
    ficeOKtuRb1yYo3Z2jwPBHyVFXuKwKUDM0xqEK7JuG39Q098d7wF6YsyyAnTEka9d58XiZXM==/JuZ"
  ]
}`

var manNoSignature = `
{
  "loader_name": "",
  "entries": [
    {
      "name": "mig-loader",
      "sha256": "355b34f99acaeb1ab18a7e08b7f19d54b7915fb234777bee828ed5e34b30c9de"
    },
    {
      "name": "configuration",
      "sha256": "6dcf3bc90eb5c76528edcb1f71bed7dccb886463a5c0c19b33bc1e8d25ce82d8"
    },
    {
      "name": "mig-agent",
      "sha256": "ce7e7588073021d590ede399dcf31f9148da969e333085a3e58ef78f9d8c31f2"
    },
    {
      "name": "agentcert",
      "sha256": "017525f2f851311e9b0e26a139252c13b186a6507206cbd0dcc1ca35258b9566"
    },
    {
      "name": "cacert",
      "sha256": "215394a591db4dbf2bbbb17a4d45b5bc6d335d15a7d2c42876d4b27f8269bda9"
    },
    {
      "name": "agentkey",
      "sha256": "88df8f032916dfa0ae6c4778fd2aa2084c1aac017aab70f7d4bc6f4327c5c24c"
    }
  ],
  "signatures": []
}`

var manDupSigs = `
{
  "loader_name": "",
  "entries": [
    {
      "name": "mig-loader",
      "sha256": "355b34f99acaeb1ab18a7e08b7f19d54b7915fb234777bee828ed5e34b30c9de"
    },
    {
      "name": "configuration",
      "sha256": "6dcf3bc90eb5c76528edcb1f71bed7dccb886463a5c0c19b33bc1e8d25ce82d8"
    },
    {
      "name": "mig-agent",
      "sha256": "ce7e7588073021d590ede399dcf31f9148da969e333085a3e58ef78f9d8c31f2"
    },
    {
      "name": "agentcert",
      "sha256": "017525f2f851311e9b0e26a139252c13b186a6507206cbd0dcc1ca35258b9566"
    },
    {
      "name": "cacert",
      "sha256": "215394a591db4dbf2bbbb17a4d45b5bc6d335d15a7d2c42876d4b27f8269bda9"
    },
    {
      "name": "agentkey",
      "sha256": "88df8f032916dfa0ae6c4778fd2aa2084c1aac017aab70f7d4bc6f4327c5c24c"
    }
  ],
  "signatures": [
    "wsBcBAABCAAQBQJXthNjCRDaHVHlrMttfwAAxaoIAFq4OSBL0kGGLqVhobmHXqqxLvEbOhdpcT5Ps
    YMhi+CMzE429mPatqB2PukZrTjg9z2dJgCOreGMk3PdeEG3HttfHZoXKc73jheZiwewSXGkopWxlBN
    s35t+FqurpGggJTJ9M8MtK7orCxFE/ei3AYRu6vELhUz+0A5EpI2Fwuo5stiGQGxpNQG4QhZIphbZn
    g7PThb9Y2f1WkwUoyiDRmzjnrDCt9XYXbrywUSfWawWMmBW/Qmq58IQNmxWGK3mZq1/oQJdextS24J
    1LJeuCR41+EcqsLS9P7ujFyQD3fbKOPvk8krzFdVHpRafevXwaORc6j2hO9vVV/kgzZ2Ym8A==hdtT",
    "wsBcBAABCAAQBQJXthNjCRDaHVHlrMttfwAAxaoIAFq4OSBL0kGGLqVhobmHXqqxLvEbOhdpcT5Ps
    YMhi+CMzE429mPatqB2PukZrTjg9z2dJgCOreGMk3PdeEG3HttfHZoXKc73jheZiwewSXGkopWxlBN
    s35t+FqurpGggJTJ9M8MtK7orCxFE/ei3AYRu6vELhUz+0A5EpI2Fwuo5stiGQGxpNQG4QhZIphbZn
    g7PThb9Y2f1WkwUoyiDRmzjnrDCt9XYXbrywUSfWawWMmBW/Qmq58IQNmxWGK3mZq1/oQJdextS24J
    1LJeuCR41+EcqsLS9P7ujFyQD3fbKOPvk8krzFdVHpRafevXwaORc6j2hO9vVV/kgzZ2Ym8A==hdtT"
  ]
}`

func getLogs() {
	ctx.Channels.Log = make(chan mig.Log, 0)
	go func() {
		for {
			_ = <-ctx.Channels.Log
		}
	}()
}

func manResponse(in string) (mr mig.ManifestResponse, err error) {
	buf := strings.Replace(in, " ", "", -1)
	buf = strings.Replace(buf, "\n", "", -1)
	err = json.Unmarshal([]byte(buf), &mr)
	return
}

func TestInsufficientSigs(t *testing.T) {
	var (
		keys []string
	)
	getLogs()
	keys = append(keys, keyValidUser1)
	keys = append(keys, keyValidUser2)
	REQUIREDSIGNATURES = 2
	mr, err := manResponse(string(manInsufficientSigs))
	if err != nil {
		t.Fatal(err)
	}
	err = checkManifestSignature(&mr, keys)
	if err == nil {
		t.Fatal("signature check passed")
	}
	if !strings.Contains(err.Error(), "Not enough valid signatures") {
		t.Fatal(err)
	}
}

func TestCorrectSigs(t *testing.T) {
	var (
		keys []string
	)
	getLogs()
	keys = append(keys, keyValidUser1)
	keys = append(keys, keyValidUser2)
	REQUIREDSIGNATURES = 2
	mr, err := manResponse(string(manCorrectSigs))
	if err != nil {
		t.Fatal(err)
	}
	err = checkManifestSignature(&mr, keys)
	if err != nil {
		t.Fatal(err)
	}
}

func TestOneInvalidSigner(t *testing.T) {
	var (
		keys []string
	)
	getLogs()
	keys = append(keys, keyValidUser1)
	keys = append(keys, keyValidUser2)
	REQUIREDSIGNATURES = 2
	mr, err := manResponse(string(manOneInvalidSigner))
	if err != nil {
		t.Fatal(err)
	}
	err = checkManifestSignature(&mr, keys)
	if err == nil {
		t.Fatal("signature check passed")
	}
	if !strings.Contains(err.Error(), "unknown entity") {
		t.Fatal(err)
	}
}

func TestNoSignature(t *testing.T) {
	var (
		keys []string
	)
	getLogs()
	keys = append(keys, keyValidUser1)
	keys = append(keys, keyValidUser2)
	REQUIREDSIGNATURES = 2
	mr, err := manResponse(string(manNoSignature))
	if err != nil {
		t.Fatal(err)
	}
	err = checkManifestSignature(&mr, keys)
	if err == nil {
		t.Fatal("signature check passed")
	}
	if !strings.Contains(err.Error(), "got 0, need 2") {
		t.Fatal(err)
	}
}

func TestDupSignature(t *testing.T) {
	var (
		keys []string
	)
	getLogs()
	keys = append(keys, keyValidUser1)
	keys = append(keys, keyValidUser2)
	REQUIREDSIGNATURES = 2
	mr, err := manResponse(string(manDupSigs))
	if err != nil {
		t.Fatal(err)
	}
	err = checkManifestSignature(&mr, keys)
	if err == nil {
		t.Fatal("signature check passed")
	}
	if !strings.Contains(err.Error(), "duplicate signature") {
		t.Fatal(err)
	}
}
