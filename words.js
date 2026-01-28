// Secure word game module with AES-256-GCM encryption
const WordGame = (function() {
    'use strict';

    // AES-256-GCM encrypted word data (tag + ciphertext, IV derived from index)
    const _e = ["n7B/jyenE10WFkFeiWbqAgETnsXCpQ==","Qz0HIJzLkaeraaehjC59KUHqa6t9Pw==","vcR8CHoOlkL/o9h+mqHMnP+8LJY3Iw==","IBW0+/PibUEh5/b8hogM0bi4yntLOw==","O8IgkP1sNkk3sbhrBnWo3Fe19ECU5w==","3pNVVfOUaxl8MIP9xSkF1wRLVXviDA==","ArLHG9jXCPfuIpGR89yaPklhX46K6g==","6d03Hz3VAE1jC4o6s9k/L8qNmvVlCw==","o52rGHA2LcSOz5tJEvA+U/Fs4AsKsQ==","KISRiduew22W9zVVpbqLBDCv0uMUpA==","Epsd9cjYDbJaCvk1mqAXpNzWelmiFg==","TsuzMrPxx1uLpjpRAPK6gQYrm//P1g==","DOfW9XdAdm3x50yfyEzB2Ffu8Cqo8A==","Ud43WSw/kR3RS4CMq8DaGan93/P2Iw==","RsRu462HF9RvRLmy/p7vEOGmR+x2kg==","u5SLX9czn+pd2P+ya7ZGHxOtXsxJuA==","Ncu+/gQLAbQwtySkbpcQ5v2q3strvg==","7CFiKib+TcBJcRJkCSm7+RyNTQiZNQ==","K5B01KM9GiL0KyCWK6b54msDhIJ0NA==","f7icRkiajuW0g5n6+a5shdgvdzn5kQ==","w8ydCLwCYeNhqwp80OiO3vbBUv5sdA==","9wwVQGdE/pdzbRgNwmt9tgFvzB70bw==","Mz5iLCSrbPwkwD1MC0BpvOnKQCBIpg==","mMKl6hSsNK0tbzWEKpItMABmM87EiA==","VOsgifkB4omZf4SIqb1djO4swzlWmQ==","/0Fj0ihy0ceHnlERAsEMRjYr2u3wqg==","dZnSmwci7kjpX2N8EeefiawYTmprAA==","qCC+7G7FuvW8BQQwXa8+hpQhl5Xdkw==","AxzuIfwrpQSwaJlrK+rI7abKoFMmdg==","fcHZqnLFm5NTAwEWVkj3yHOGtRUwiw==","dBSIWDzVMIK/IiMJjXdvgydJgVR2dQ==","2IDigviQeucXV86nSZMFlYHiq0Ge5g==","pVwvjFyO8z17zU3bLZB5vkbf/81BxQ==","wmg1EnE/ZZmaGvtEQjaGNa77GPOv9Q==","aGeuGuBToDiTJ2PEtJ5QOAtMJYR2ow==","EVDpBMz7IC/Orq+TH1lLb6nCr5Ab9g==","MjJMJ8G+/WK2UUGAr9on1z7FTyPphw==","XdSfalCQzxGkYf1qQ7w2UPs1tV/iPQ==","X5Zg/RvfwsIoU27jixpaPt9P2NBqyg==","UTzAFMtUdGyNfQ3CkOCfhw0zT7y2AA==","cbHKXovlbYIKt+ls1C+I7sytYfotwA==","8yVMlpzrBbBCgNDaAkkMUFqSrhrIcA==","paNQrHhWcn7/6fdcsKmSEl/ofHaaEA==","U056T8uEz3PNEwsLu6EbCfMNRwLptw==","JwgvyVbCERihTQd9t5MT800meBI+4Q==","qcQIw0ogUgJrB7laLjuUCWj85J/PFw==","PhRjjxfnGWI2P+5qa53NZIndEEUE0Q==","iQEA6k5pBr/tRCXOZF4LXCp03I8qyg==","zgOVrF+lp6R4Wl26XIhmVUfD6zB3aQ==","QWg5RSoD1qTcziTalSIEkLSciRXXsA==","S7KLkP3Ya8bbPLnXtkmOTrFxXcCixA==","HtqXqLCppCzrDO3FlLKANch2TDoFdA==","UcEs1lKkHfRE8Gn6lbwqce+m4k8LEw==","7wY+oPzTpRur9bGJgUkNUyAGFSX6SQ==","whaKkBGFquam51QDmID2SrUw7F1ldg==","Q6hpHx1iCmrNHZQ3KNoOOdojk0OhyQ==","KQOTWZg9knjHCkSSbHfXERqcHIV1MA==","l91UQOchOsfVwxd3DQcDncd2hx42sw==","pdJB6TDcksmj8klpXIhf3ztnqaqmeA==","g534gdSpRUqCEZ+aekelHfXginz0Mw==","a+EL16YyJG/ZZ/idh8B+3k7HmPvdUQ==","wiTEYe472kaZwDeSf2boTNAdpcXUiA==","aII3DiHP/0R5khlBRzhcLx4r9jM5rA==","oPuIqRGNpjWMLydFF6QxURO8soX2Bg==","KsCyS8wxyqOT9Gw9DJwS5WlX2pUx7A==","aqRYbxviYH1oLSUGCXiVZTWnc72a6w==","nuvPQUYDzEVpE72Sia6TPh7c22dP6g==","Mnojz/XIoFC7VbRq9gWrrV8QnOEsFw==","4F8RWmZRUxolvpGdOdhPFafNrV8xWQ==","5nFtE66WyP/4zy3CaCaLAeccgf6lnw==","aqvEbvRfHUfJUZO7/KhbuMQyzFNCKQ==","yJ8q2+l7wEF7i7ilNSHP7E8JVwmrag==","KeSqVIF+BlT20M2RCYHbKf1hH5D/3A==","x7+laHJ3kaFECpo2Ly4A3X60ve3N/A==","oer+prOslyNQpHSoHcve93imQhD7fQ==","W7XrYatfHjAbBfr0lmyD53Yp2d0XXw==","gjrcEGrtcHx4P1tJbbpqaBFdWq0yiQ==","FsZBeWxGQ5NPMuDMNPTE9j+TAdPbqg==","n244yGZb/xKWS3MG1eWp/urThZjfyg==","pIud75D+mrRTMH2gNwjr1UNaO42Yzw==","l6akmvjBZyUAH2NpimUFffWb5K1IpQ==","bGE8KMYYAIPC8E+Au8+GnKMR5XAftw==","pCgnFQnXrGB6fSV6MMDHj9SZeqU9xA==","+Bidv3oiqmSVQq8w7sPL0T+xO/obig==","AwMI+/tCvkp1FNxi+6sF5mDsw3r/aA==","v/qh9DGPYGDKc+iwd1l985ibdamYjA==","Wf4IEub/RDU1LhvEgOtIQwVpNWbKFw==","usIS3W69891AiK6a8OQtvmxN8c3Y6w==","Z1/jgn3Yk7RFTiywh1SSmaG/vOzlhQ==","AJALNgk4Z4MgAAhHnWt4VCvwOcrEkg==","PRmQmqc4UBAPenvbkrJVuT2O90dEbw==","ikyRMT9spMW1ubL5Ih2beeiygd7wBA==","szKBnq99R+YyQ+HjsKhvvLD4l2iuTQ==","PLoEk70HXEt2wDuq+2NEa1Fin2GyCw==","VZgJJ2pJ3JamrPSwKtGZmQqwDJjCRQ==","yC3Vgx3s3FhAcQ9H8hNQ9fWPZ3Do2g==","8Fy223ClTu2jEP7V2RWDg1wVeGnYsg==","tU4k00KW23B+oAKsPrCtieB2C/hBAw==","hf8+LqbBdD+QxHGUio7PJV9UXK+/Rw==","tf3DhQpnbdayAvJsWyPguMm2Q23jKg==","Lsn6dpWs8V3x9flN347slR91SpZnDA==","MgrX+IT6QKYLbw0ehz6D/x04mMHC6w==","1JAVDP3A8IlXJwRZGhgKTP4VkYbASA==","ot9DVecHt8XJCIHN/tV42PJjOjLo/A==","gyxkm9P1Dg3ShZt8cS6IDI2xJOa5BA==","BzptEMSltupqrq3/d7Abtumt6wBStw==","p3XJNu9ANFuZiyBC93v47umEpxXBRA==","pOGoHtLSxjkoMWOTVhN8JQ1W9fh5ww==","Ay8JNif628qwLsc7XEnsjl5Y9ZsSyw==","Ak99MacqNSOMnjNLX02XnL5nCzTNOg==","0Uf5wVhNYtZJ/1l7UIXx7DXLEF66Rw==","oo2AAPmlE8nRm19yZxn2flQ2J8CS5g==","pcqQNwCS+KUScZCVuEodSf1YzwLZFg==","2llc5OEvrVD6xmo5TOzM//6cRe9SPg==","CwU1q5ywqX4BoBAprIaZeSa0RciIpQ==","vvGtNNlddX5e/Tv8kNsH+CACNoBewQ==","aM651s1W550yc87pycdBYAmHsFD/pw==","VMNSIMjGwNsM8/zmNuO+UK4HUqG8sg==","DFtERGB92tkZZ8uV1PwvEqm3iyICow==","E+e7lWCvMQojFaWWC1j3GRHrGpUqEA==","e5Nfl8cukGCanCPosPyOhNYphLne1A==","HFVMi1838byPUKkV1kJKDibsMT+X8w==","IzJxJjLc+GxHKSQVoXBV/MulLK5L6Q==","FgaheMFnK30qkqvHqRi/cnx8z3Suvw==","2Z50KN6VZR7XyX0+NQRcEA1PbUjcew==","CyxqN39BhLAw6VAT/vSF/WV5mnXrWA==","nIzyrKskvCBVuIMH/3wDheAXXXq5FA==","tNzy8IxashKOWn5Yh3S7xwQB3nj/1g==","H9fecD/haLSxl6jUrEt6Y7n870RLkQ==","hcXSF7lnytQ3FvXkf+u7RS26S61LSw==","UDLRuN/xS+YtXHASIuU3KbYO7sT/3w==","keD8f6fM84/PWB7Hh/S3WWbRu/Geqw==","SLNHXt8K8mbanHuQ5j5JJ3gyn3bX8g==","oyuQZa2VGLxWPCvL4jb1+AD4BHmQVw==","1Nt3YCWcPuOmya7ntiKmQ82XeVwuzQ==","kPtc4+OwbU+294ExxpHRYWKrXIFZPQ==","0jbAAdXwhUnmi8RXpVpgz20TU966pQ==","TA020XYD2caKnd2jgoW0XUPCp3Tv3w==","LXLnSZzDNY2aBwySZ0ALqvcIzpCJ6A==","v2SgEGz7m8YLob6zETfNz/cTdpZvGA==","VoUUaWT5SHUjUE4oA04VMhKazzQWag==","5jdTio79caxPjCWZ8kJ/LC1TPtAKjQ==","ZOOt0cYaXdCOBUIWllGhMEKm7BBUAw==","avuCakQOnLhxELFuth96IW9fdEGyFg==","rLSDSefsdI6+LV3AKaCxLyWvh+3znA==","qad33ghfSsbjY3qoe88oi7dib9Ht1Q==","v1tBcfURMpOEzdxv6YsDCbk85bEV4A==","PA24an/p2vvrr+76fXeDCS5MwbjztQ==","lF5hBwIOeeluwgiMhhKtOBc0PtUuzA==","G4lmqlM+LeEWoxfmCakerrVmDsBapQ==","Bu5mltBRhsqaCvlKGgiycK8IcJVpfw==","V4dgUb/+4E54BcFDdsYFIvqxYy5mXw==","n1egaxVdKrMbP/vV5lkrC6i/APegMw==","C7YNxGsAGuZ/eh+8oKp7SsqVHAVRDA==","dfTS8vUJfYV7z0pXTlEgnQ65f89+bA==","AAhC63hu/3v5/m424XwGEJoMXsQeGA==","bhB0YBQetIVdtnMRVEPZSvCL+FmhTw==","0AEyEvn9HJ7VtWCS0cA4H4Oa/f9TxQ==","/IG6RV2kwkOxo/mipiDg+PEDpoGU4Q==","cU3aYDay7nBqaeuvKV5c9dM5F0Kyng==","yaW2kEutT6GODgTKaB5/6KYZ73jdVA==","Bs1G8aDu3W27kC1TxznaUPOdz+XMeQ==","NjWEqIU0SUl5mH66cf8wFrqxIoW8Hg==","ldiNBCAExGRMMIbkaQn75FOWvWWEFw==","mua0cFDnxGOjYQfsOMBy+YhZItcfLA==","pnHQD3SmN7p/CdSmiUoRvtrepRtDWg==","X8WhVlDgL0674Hyzm6/+DCRtcQ/YRw==","0PKFgCTfi0NIunpxFhHanN7o24gAvQ==","LSRwcQuGRq4AiXJiJJvGFTEWJNduMA==","8PrBQI7HWb8O6XqpJ8y2cvBlMETM9w==","/8/g9JA7hYo9+wHo8hhTVG97pvMLnA==","YTMuQw3T59SY8anHAUSRcHJklyrh/g==","chlgWl6Dba+qyL4KUNE4cbjhduqDpQ==","//yoV8I7J/u4Uy+oq9NUHziWH2Rysw==","oto8geKsZovkkO0tsHO5dB7UZJQeOQ==","0Yc+ckIvJJr5qMZrNlddzyIL8pdX/w==","DRahCbzJwq9+zc7toOnpkaaCaRukPQ==","hu3dv6c4R8NjL6Up/SwvJcNY38b1yA==","/oHiQfuPw/5cfplF6YDNggkncSQNVQ==","hA06qU7UjPS6kbN8rbGNMIKhIgWC1A==","aWlaiykIYdh4qotgQBALtomb0akjGA==","duz6jbDnXDPZav85toguXC+LvSgQZg==","4tkAp+ZOsfAJFRV7SyFXD75HVy4jzA==","+1r1M5/Fvw9oDgQZQ0TdGt+XlGT3mg==","PKlUtf9MIieQTioPOopxcGEgZE8eKQ==","cPP4hNwpc0+rnNhffmN+E3XL3WQmNg==","L32BWHl7X967+gM6nmHIM5sUBX2dVg==","yafU3CFA3a3JXAdecQJCDAxo5Vcq2A==","Amna1CjOFC76Ygk0m3Xjq/E87TpJjw==","qO5t4taV4DSzMMaFzkmaAzpLL62H0g==","KOyzKlSOG9pilTS3qkEmjOxdgF6drQ==","gISFqz5nEaE5OVnGxV6NyFiJ0M3acg==","yk2rzanGrcyjX1+bdKNbEqmDHaBqcw==","xmXvPa+K+1xAmMsBEg9H5+gCtz4Knw==","mxzwi8jHYvUypVgu4OJAelvm2U3yJA==","7InP3HHIYB4f47k9eJYAMuyD9SFapw==","GYz0i/wuU/XV2mR5dx+gWyXn47aN+g==","kROMBsMeIJ9e/i2JpamyYZO+IL+rNQ==","G5FSiXZus1pjnumqWXLZSG2pV3BAfQ==","rYApaQEGmZolTRReuVpiWlNoLMqONQ==","jmGFLb83uzw+mAbbjIUeeDbwCJmaYg==","kIBqn2ZU+MzJnQj+9lmW0U5juYGQNg==","TxpzT77J8PCqx23cx43cNUI+k82z2w==","bkkkiird3IGNiV9c/mBsik5k/7FKOw==","NKYs/+8olwSeY+yKXwr+Kx9miFjmkw==","zay22/d7uU867K4HLXrnB8jXvD+SOg==","WLd0o7JvXwFjxOfGudvyWZUKZcq2jg==","VUbyLnUL1bJTUN4HEKTx4VTZKgemzQ==","zJcWR77Gdul7sTe2LTjZITPwckA0jA==","FAoEct91UK8OYaX3ERMy8Kil35+/Xg==","y3bt44WIbmrjfBexEqbEPf+EXlFirw==","iSCCw1O5cHRq9ysQdD5fCTycuK9/8A==","LgzgGc9YgcQGs5CUM1AlBk6si5Rj1Q==","PfaCeKI4+wPiwj+r4bNTv7gvZ+ux2A==","S+E20iycZP/iLS0iHIJFo3+5NDmx+A==","/pvtxaIXC56cGYi1kQcrf0Kddg71HA==","K0DaycGUQ+Bgf1NGu/z5bEMaCDmyrw==","xB5fTnYh2ngVyTVvd1MT/0iQIXE3Yg==","Ps+tiviSAwAqFEFCGeCi/8yqXy4ceA==","EagKFVF6B6kWVONliRVw+WOkPROUCQ==","cjaKZjutKwLLGlv8eR+A6z/BARyz0g==","OE4rIMdabwy5yF9rCPomG7sU5n4NsA==","1i9eBUcA+TlA69pUR1/sQ+6u6q2bMA==","XpG0ij8TRhCbXv6rny/l0BlAksX/Fw==","9ADgglhsf3IddvufyF6278xgfmk5Og==","mOtl6v0CI1zmc3FFwM/oQWez7UhFAQ==","qT+ZLsCiA+6niZkbSz9ONj+apWo8Aw==","WP5mwo0zvbndhxGle+7RBXEAy6dWVA==","TScYDEd7CZdA2TkuYjjECfX0YSBWVg==","HXRFvtdbjGcW6JpaupZWV+oGEsD9VA==","Qi40G+imt9HlP8uzCGRb8jkajhGgHg==","RHge8hn2YrQAoxxWwlczxjKmKDbPeg==","uxUiKrsy2qC8TE6yB5nxuftEYQyrEg==","oJUx6Qt0vdq4zfTaxXBu79BNWPWT7g==","vEZ2FuIfdORTvCWig7ddwkgB0syC+g==","mHclRYF+qYN9lHbzuuA/Jcuhv7imQw==","IkKmVoQfeEFpiU8B+3fi4wgBWQt52A==","Ae+QrRrJ5ZpX7tlTM4Q3/hmN+mH7mg==","j7ZoynID7IanD47zBVUgDYJERypGWQ==","lM4brihoePmFS0TwN8Dm6X2RNh/3/g==","4fkAUtm9UKKZx91ruuBUe1CKmwaHIQ==","OmpENaNNZTXpbkhOricNUx9pMAZQRQ==","DxbsjCtg3VWiCgqLFA5hJChJWXXs9Q==","YinVvxWwx2d0RXFEb5pPBp5yuP+xTQ==","Mq35HXdKnSzycKxHEM9Y24yR/XqvNw==","Yr5iOzztT3WkTTVyBtgPmcbK4aD80A==","Gd7RJL4J3U/5cGNW2isimbzmG+pj0g==","hzlMLOKOeQKqa+XEbQkJ0WO3nSMfMw==","QG/PSk1uFrY4/CdSZORqjG8Z7r1q/g==","uymjvC9qhyZBr0JjLg7RovaZK10iXw==","83cFGf7XFSX9+v7X0wzOYKTK2Tf7gg==","DsK1AWURoAJcsBQVHHM/2l+JdFSLJw==","HyRMBbGyHmNWIdQNY0yzMBynza4Q7w==","BxG2gkSHAjV+eafuFBM2fIgWnYSzgw==","bm/n10ELayfFuPM9lVPi2woFiuZ8fQ==","dRutyYGeu1G5dbkqtDqNszRUmihIAw==","5iDkMSleZOUqLRbmdsu4mdD5D2MdKw==","kED0T2Nwtcszv26dVjCpVeYbGPSo6A==","NA4xOos33Xd58WVPtYAhhFO+L3pLpg==","BSSQiSFYyq3lW4sT4LrMqS3kU+ScIg==","vNqHtdz3eCRUZDAiUw7+FapTylgnEg==","hyfALzZJfpJPT+y8ysgZhemWxTIPLg==","Oo8qtleQDXhbrWhXB5AaaqDfMZE2Yg==","Wb8+5Wyuc4i6Rn7gDZcFpyECUtW8Yw==","QVxlKBLnYrY0nkAvHrSAuhqhgRLBXA==","7UolLhrp+sa/ykB0tEqFPsyKJzCJJw==","Ay6hl7p9J1t0zCd8vEqJJsVmckxziA==","/2rWdsrNnQJXITzJa3ZTIieVjsJyAA==","lfgtj4lRydwkckfE//dxY/qPcIDB4g==","qMkBR3qMs90W/WizgSooGSKbpNHkqg==","vE9X0kRdaRwA4hC4tnA22dAYeJZ++Q==","+2BlAxBr1YfZwXC7uQCj0TzMp4IM/w==","zVDAwaQYzKIqpSFDIbpSysrxCXqMTQ==","vzk7yoKHSum9dDov9EqXT5p8FbSLpA==","7tVKFlKbEk9LeYH//9/SEPQaV29jhQ==","JjzWb4dcTq5Xn8vq3ZoVsD9KeK9AEA==","5bzjSDa2Y3REt6Bda12Gw7RYlwTzeQ==","X7ETVOU7fkDJEi499CdMkZD+EkGOfg==","nTjSFVGmL5qcVxCXsnWqqoM7c/B9Aw==","Tos1qaDK8r0EQfRR8dSa3OxjKkOcKQ==","VNosnajZtFOce3ScduMFeJOo8HaYGA==","Vul3Jk+iF1tBuL9Oo+KAgP8Jbf7b7g==","/u0kcq+frZ8sKvWR0HMPHTp/5NiVBg==","VXSjiCcNdlh67GjkhMqinCSmbaFmUA==","8OpiMWoe4iziqn4zbEer47E86doA4A==","oSxnq3MYsQc8XdkBKkyQOMXF3W2BvQ==","JdpJprusHbzBU683h8mKVYI0Ttnjrw==","XSAm6f1JoAYsXwBBWrbFgPKKUrKZ4w==","ZbX6mFCC1xPf/P+RjCASVEjBFgFA4w==","Oy5TMQ3xPgpx1qsUH12TjYqqRy24+A==","dDS909X78cExX7vqEixj8OG/vAP/sA==","cLBXKbxTqZ+j2REbwQX/WOKoHRqF+Q==","qNynAajrHBuu4HRQL2hLVzvG7l9vHw==","cPQNE/U+lCma6qAL0TeSoaGndJVn9w==","CltVHOu+YHEWXDwkd1k1dI/RcO/umg==","uaVBgTpKositu9ZkG3LQ6W75No72Tw==","35Mdm1wZhWlYyCjjEAE8gUfz2F25WA==","atRI3cKdoVCrzKIUYVHTUrIfOtsg4w==","YERtBPyLtefSqaua5UnhwJRXRO7tOA==","VdTimJeI9tbZJlRzLQki0Qp3oHTJQA==","+zGt53acyGzZm9SlmrCF8/cs/I/JTQ==","udJS68YyPfrvD51HnJoV3YHqGV8/CA==","lCz040giuIuwjX125B8qo33nDoap6w==","t63OtWeEoBp4okFv7KsRyMtvCnS1PQ==","MwUpp3ShpJa6zYY71oPIPxQkepH7Jg==","Gynaykj+YPE5DUv43mj0rU8IQ5+PzA==","jnndbmYb31QhNr5juc4x8aSSqoBVeA==","e4VV58rhBdtLcljkPyv+vEgQJxDEOw==","/MzXO3u0q6kIbXXCATsrsYvBYfKCZA==","oOZipK+n+PT+6gTFN8BNsKy1z0pdhA==","QueCjlcc5AhZGNLNtiHvzLlUW7o+6w==","5w3PKc2J0mMiuXDJmHAGXMDgLRDNHg==","LAFvvFsXZ72q+HIgfnqg146O63H6YQ==","o041rMTHIssm7pGl60oeuPDuQtaQng==","TeiEKTdmqUmE+XDkjgbI3Zp2t6SKjQ==","eADQ9PH5bYXbtMr+TtFX88CA4yjB2g==","L0D7RY8VFPhcllOKfbbHIx1gl/9Pvw==","C3as5o4Hm1FDV8AEMBEWKG6HSHMXtw==","N8wTn2wmMd+i/sw1XxgApyC5zVvBRw==","oksIjg1NC2e5EhTzftfh5D/jeSQnSQ==","8/ReAVZNZXyoa/risBTopy09dXGKjQ==","jlNxNiR7fn4kziUVlAVF5b1wDelnCg==","DOYZUZScAgVm17Vti/fO8eERvPuw0w==","H8P34h+1QsF6FC3xFL8ZRgqJz1daAA==","8Rpw40xvlsvzLtDxNhXMTJcjyL4LXA==","beccnmY2tpslKzJFHwrAmxwriu1tZw==","jw56VvctodX2PS+tZ7ik4YpLK24IaA==","JONH0u6l1EQIzI0R4TZdUXfxD88LDA==","s+TD3cBAEnoHgAItaIvw7akfu9UPPA==","xxTM6V3kxuWiD61v5Z1y6WkHo40RSQ==","TP6y7M/FTr1BouJzO52HQUhCBA3nzA==","xDn/AdlB5KIVLwgoiUkgxPS5VD0HaA==","bkdpzK3Hw5mW3DjIpErpdajvucEE+A==","FUY1s2aCwTRkr4LCmQSeo/6TEo16cQ==","tzJrywfGm3O2G0hbNq5P/sYdMpx+ZQ==","Z980MW4Zs+LclQdhPj1LSFaFynhq/Q==","ZB2/YRkxlgvAfOT3rYIMS/C0SlHOJg==","NPQRiAssqjE4qtVK4RjRKKn1kxVk+A==","IlnQpuNERH+bxMRjMe8w9QnGV0va6A==","fHQTO+w/rxIOB19+AJCjIY5CrS9jqQ==","iL+cHwpbNj4rBNA31ayyvbkX4M1/6w==","D7mkYhUxZxHM9GpHtDGGKUYkA1WB0w==","amvxXZBSw18XVwTpIEmF0ol13rFfGg==","2WywP9cen3ZdS0dgMCUiBDqN1bjlxA==","SJvarMCJpJhbuMprW4WKfMJwMnpuHA==","NRSrNIm9E6KMtHQyYzsLWjQfsui2PA==","X3zpzwRttOl42GHu6nqnzEEuTHwZrg==","aCIetT6U51q31Mr6Vmz03L/TAybkwA==","W7w1yR59F2/yY+qslPrZ9KLH4jAzKA==","XZDKpPXTqeqMDjlEwLFRvbEhx8sHJw==","lDZKkk9I7423J8WSGHD0RF7CXcRqpA==","snWKmN3CobjIexuI4zN6xJvtSRiVdA==","FbT35fDnYO+NJ7IDHsJ37PBoj6fINQ==","nJ1ALmO0SrjSWDFoXaHee/hHpG9CHQ==","5zbC1xpzKAH5Sv1/tutmrJeO8KEwjw==","FoK4hVcGYsEgo1dvD9EwWIO2udWT9w==","+5POsuvs9fJCey8Ds5QLzD9sVVzQIg==","l406ACQ2Wc9fH5T4HFLgW0vErF98gQ==","5iHXnC9TuWgne5spogz8CUvd5FP8Wg==","98gjw5u4mHou6lsOpoYhqngEuFDGEw==","An6Tj4loH0xVpHoIaZHsqhD75d54vQ==","EFfPcvYAlS1V+CHSEJw5UAjlANGqAw==","0/VqOBNaVYQIWd5jphUhauYaS/yFpg==","CQ/A8hxrWvEKY7pFZDLG/alGxZJ0UA==","v/LGntjOPDgkq/DmyLackT/2mfYW8Q==","FH0yaaF40RSwjAfo1m2mmbKiZ7Dung==","lKLQFVmVha1z8M/8+znT9bSzHHa9IA==","J0XogdMfNo92DnzuX3EiEIZC2BCxBA==","ZPnqEzoQRAImY3vb8BYJlzUyn3jOBA==","LAdQnL7ypQQHsWXJlPM7FSsRwhKFKg==","OYMytHg26leimKX6jH2dXMx+5JULIA==","j0bRkgHhLGehCkRk+ZH+E1ImziF8jA==","SAXtB2FH+GDqkoOCU1QnPC5cjNse3Q==","me4YLEyBo0aifIMqttBmHMzQg2q11w==","t3xcdYDOO2rlGDlFUYHPkwbD2W8XzA==","5z9mOmwPxlL791tQ1F2RnD631CtDUw==","xnMJNplt5MwA5oKpGFGnVZuEJLL1Bg==","bAe1BQRYgFmehcRAgqfdP8G2S7E0wA==","jyOgvlZWjFklKSFLl6IUkUA9Jmj+KQ==","X+DMf6d/w8J/WRwjt2lGw3l7smGsdA==","pBxWy0leu1DYafbXlur4EyQctghEDQ==","VccJpbIAxGSlicMbtGrC99CbwA1dwQ==","f/RuBf4G10u62QULYLKZD/1AfFNIUw==","hxSSz9te75CDrRtxU44bIXOJ2Jfmew==","4RCQmFRMFWycacQtnhY708mKn/ct5A==","3AbTwVgcKsr3/ema/TPslW1o1jAi/w==","BnjdM5mTGQUefx9aqGFKFHC6EUw/bg==","/ehfiibUyo8gk6CCSNzUPhnyodyhlg==","xjESUu5LAbfuMtGmd5eclh52b/Orww==","C9AtmJ8T8zbqonNrTYaRjX/xCMHNIw==","1OIYrMDSGd03C9rM51ttSKSlZKYMgw==","4N2sNLZOhQWr+tlFLwL5ftzAkeARkA==","/7gRa9vx1CFe+UzJWQg902rxyHMQ6w==","qE6gZouU/Dcwj2Lhq8F9dhio8TK6jg==","lXI8A5IwSssnTPbgXJyNevcRHv8TLw==","FpC6yTMXtZpRRccXnrmll31iu7Ur7g==","cV57aQ33GIXrWOQu9XGfjUH+cOGbbQ==","8ETEaRYOe6WZBn9RHdWSLYB/UfHJpQ==","cc/jUQrE/QX1PygdECdlSgrWnteVmg==","qKqC3M4cfYDes8Tr/A4QltXl8Xu/cQ==","+01ESgKG8tEzhix6XA349zR4nJhzMA==","evgNNy0plsmj6SBk7OnHl/t/GbDKcA==","Kf6Nvm1tHT8X4HqI1wvd3ePZwWxKeQ==","qqOWdYFVQgfCHws6YhOTaB+Joh4jRg==","vaY2Ji/SHYKV4ZWedN9T2mUWUr2NLg==","H4sAvx7jo1ETytZBoLbbnxFZf0BqEA==","/h8zqXxYdfcYp6RhITLEbF+raiE+gQ==","ZWWcqqPRDr9sN5o4MZTeJsP3RHic5w==","AF7RLdTBTpyIMjIE25uJ87qspkRMgQ==","9vp+aol1+zJRYd/SWAmI9rF/RfTQcg==","vPEcI+siy/jZiIi+CVRCcKjuRr/WQA==","0nT3JcYKucx33JQ+ZNTwufv8+MuHSA==","jns/ybxRIsi52bx7Js31fSuxYGsr+A==","qqrXdt4W8UyAJzMJ8fN7JWq4JVrMzg==","k+5snXSAWbWbIuc19Dprq5VUqgaV2Q==","B6RhlykTK888G1vY8S8Jq9j+lPZlKw==","+KQO6mIDfTgYZUfdnL1ccVUvxbRxVw==","5jffTqe1GGXZLjQ3WtW1jjNGChjbDg==","ypolOcf51MhtE/cvEeY2940avogSag==","m3cRu/ehyOElJu/WX3Vrz4hoUbHaRA==","ivdPOuLzObgt9O81txQtN/1c42dG2w==","EPCpg4ZOpkmVJM0mX34IwbL1qiZe5Q==","oK2XFJOopy63djgqUoyVW/aMztDR8Q==","68CwBwoCC2TG/bPiFnESpJ2RwJyxYQ==","eu4YQbDwL/h3lVxBMMjlIbjrUMVdXg==","0ztVRBJ7gCLJFCMdTvPowWcMaBeIzg==","6hzj799UzwsH69aN6alsJwqYwB8U5Q==","Dfsy5I9CGiOOW8w8iZ3yqNdTHvX5GA==","59hEl9AogleJn7I9aZaSnnCm5kgWfg==","Ie997bzvMgJC/xUvEFf74KwDttDzkQ==","EDgkKo0Xfu3wfO4Fm06dyo+JKUNC1Q==","rJhA8HaiA0Lyeb109GC2Q5sRKsfNFw==","iy//g5x5HH/6U966Emc88nAKB9czmA==","D10l/EL+zg+T1+sPFGXHoMLFEvJR0A==","V9TL5ZKzksci+lrB5nhIwV7iWPLKTw==","U26cA47W2bU8kAvRsDn+EqT2pkdq4Q==","yvY9qCAvG9wCZ1NXB53uqgHtzRMfaw==","70ZBKpmhPkee9DPKcEHWqqo7AffBFQ==","8rfS4PltegidOqDvT127H0NNI1uF1w==","P4Uh/AQhMYXmfGKEElPs+XVOpeZYog==","6A8liOHrUGsWnTMX+DWRbjTm+hblEw==","HfnhGUcn6IJGYZbCRc1zhXYLol02Fg==","huVK6t78YKdfh9CqrujqDSVAPVvbkA==","wIJs4v545f0GZWcD7ExO6K2vtFn1IA==","PQGiQoxJg2MGUfSq0fs4VvFpGYDBiQ==","0jhCsAARtqmlhTjxQDu+ovvkQEFIIQ==","edabOhOuLHWTCirOhkyev1uRGp4W4A==","29N/XbTILo2GGSVMUWoWnLzPcqQBzw==","lj0bJECRw3Ys9dJDOas1NasdI/MaFQ==","B7nTWwQXmVw6q1mHFFWp4ZJllxIhZg==","8REaWmLQDlSdRcLQqaGks8r0jKbj1g==","Z19F57IKJZnGpMl8ebYF1Q+CEhgI5w==","zc0uPDx2OQFSw4MBFfIXYN4FI3SUkg==","+n7nRpSb+dTOqj0+R2HCql1bNQCbhQ==","pg8PxoLcXsbdw965PfbBlPcn31/QJw==","S17LVBxYnd1A1QSU56fSkKVdc38sHA==","o3Y6heZJwdQv6rzuef1Fkz159drUnA==","pVc1Uv4mb5QMba0iiv4pDfyEuphe7g==","UUN4F6qLmf7xMIHtId6giryUTXP2mQ==","T2F9EFLaK4pzpFsaSDhPiDf5SaVx1Q==","pkI1Rb6J45opxB51Wp0f8MKsPRWuvA==","3YO+BbxRxsKMGjHrGYQuzTT4g6GHNA==","w4sU/3OaSES1p3puKBJQYgTQ5wHZ8g==","NY09SydU1SEm6uKCPO6g7YKvclYvYw==","F3CdUEBLtvDY3176SsZuYBRnfla87Q==","bi2eht2m/cAos3y0J9ZDkTJUxuWgzw==","Nk6WIM9idnj5XlAQgnNQksX+YD5qew==","8I34+5wHV4GBRO0KfsbF8X2FWWzYig==","2zChmJD/JqwPsfc7TO67Tv+WGpYNoA==","lCNEFi17nH/DiiGnz3BQumFCBjrz6A==","/8z02SXBtjXC/Jl9D0bAwY6u1mtKOg==","cUtOF8iyRnvbA7HYs/JE6sfRqq17ig==","6FZWMAKhYSJ5Kd+3VUbshmjusTCMQQ==","AMgJeZ7+5OiKqhaoyPX3L/SQ9hGV7A==","TiVk0HSlpJP6CWrHGBH6af1QEaV93A==","Tmt5L5hyerKt3TbFUhFGw5IAeSI1UQ==","8XkqZOTqtcJZJ3Jz2+QOAWZRSd50qA==","FQjAEqHUY3V2ibiDSb9gMhu8PCM6tg==","IKLmA9h/TF7eoswjqafLPQbCU4rkiA==","xgi6lsiBeyYJXoixmfDLJxmcrUUPIQ==","RCivKkn/IYxCs7LBdqVwUxJAwoHxeQ==","iU6nlIJpnvM9eWLHUaG1DiEUx+28BQ==","cUCdTQC1AJdl1/xASWNyJ2xnWwflMQ==","RySR2Ma6nNceZ9c1H56JngEEvc0nQw==","yHIOedNdX2UhZ4uzjNEZoU0kc8MGVw==","MVWGTI90j6VhH3RReM6eYVMuGZQcfA==","BB65AlrqTiX/2F9CW3545v8p0sRHpw==","KUdpbq8oFohorwLXx3zULn8dvMjvaA==","RHuMfEHIVYcdbPZk8YufXYaeD5MAHQ==","Nw7ZsSDMVHvySNJOr2nuqV1Eg8jwRw==","KPWoFSwymSIflBngeRb87HobZmsstQ==","Ywt2kziKePws64IfOygueSEsoasFMA==","DoKMual4Utjl7XwrL3FfUMdo2h9B7A==","2MGLuq6Bu1dxzJSFwLKs4EMThVXJuQ==","K+uNXF4vcCPCtop0G7dGD1+1Jrqxbg==","f5p2uEQeisosAHpPQYVmxlg0Q5hSTg==","k30NS09ecnV76DBCwFNHFTl7eZJTCA==","dPPoruciPzxAAQK2z50+BU5P5QiL3w==","BV27NGJtJf97RHr18jfpEvWhyXADrw==","jXADC4ZX0fkUmORElkmqreaXUlBKzw==","6OvUwiqeABNnIjJBy7FsZiK3DEkPqw==","YMUC3pYTeDQQ7weLLkFZP87Am/lnpw==","VxNQE4ObfEE/SSSiO2S3JXDNDtIYiQ==","7J5iSDHjhubzXsCywGwWRbwT/lXRhw==","viflTLdkhvvNFg0PYPL4hjqS0NBCig==","cvHi69ZcqwpazYftPTvP1OcyQ+VduA==","0N8vYiWKmBRVSxdIfzZx24leb1zv1w==","9ZGo+OO14jvPSY8DWzhJ5ebIq36cGQ==","r2COjapAahW4BUTdYUMwK2exgblAKg==","tx3XwXb9wvVi24gwg/P2byjupzFNBA==","XJCHqc6HE2PfsMxNYGzUKNJXKg1iVA==","eS+O3cPBPvNHG8T4yBNLUyEN7iuUPw==","bZXFhAi6xU15ZDY3PePi6uQUFUyGDA==","nO4iKi/nd7Kfq7VXz+u32bmmN/u5HA==","bKxShCXyYN52HGEQRVpMREc7r9xFlw==","mTKd9UDbce83T1PE7dQMsRYwiYO4Wg==","Vipd9bojk6gW8tqgz2EwPee9tlHpLg==","gY1ywt7qj2KSAcZTYifxDg3me52Lhw==","APyQsF2B7KYPsWsVGy4rxohxsuv05g==","t1V5bCjjQwtzYEX/cBdBsvtGE8P8iQ==","jjuGX6mQg+EJHs25sM564pP8Cwo/mg==","Rc4BsaOL9POrllgCMhr58ZtlX39AtQ==","ISR0WnMvicLwemDXLFIx18VTCffKFA==","EvU7z+bDjFd6uPIoToHrE5ff2XXinQ==","oASylkSYyKlUOoT4B7Hkh9wU8w7RJg==","2QCixd3ojCobdjec0PclIoNjJDFGsQ==","Yndk75rM3X8cPYN0zgOPOSsOgjOkPQ==","wLMKfBnl1fnf/0g59bgnXDoUKd+3ZQ==","g3tw8y+gKvixjagD7KT3Ie2tCHjmig==","DVIOlrAxERg8SWzMC33ZkdGELxH4pg==","Lrok28cpUBCCPreonFXNRaE4Y9xWpQ==","HVbaZWmzCK1KvuJ9lqugHmUbr/g/dg==","xmaZMqNKPx9GavglkMkPAgw9oJB8tw==","wBOb8fCxP0BvmdL386FdQnmLTgm11A==","v/qNLUnVflUUgwstmPoOM00sIJ+ZnQ==","y+iip/cEB5JxnTBqo5KbAbbSUddaTw==","zEH31S9KjxK7b3Vkd5hYnvB9xkYLEA==","UKO8RUiYLIYy4vk2yGVHZuk2C+GXiA==","QRMe8N/bHTUuy9eC6f6Nm3Nm0CAofQ==","JcK+4fHCgWrdQkBU9MzOJP3WZcYFqw==","cAqbeB+wZT4GHP5GZaZBWojv0G3vbA==","tW1kYzpNUkPoBqXota81AeVL/kv8rg==","W/iq1hvkf+mtA4/u09mJPYNMj9dJlQ==","qtGmtqzfLVTyJFJ5ifAIJvC+1lWjdw==","qtTknBcEqfUSAdwGVRE/84hfIYiUiA==","0KFHdKIRFNyQi8PBG+n6kkF0c9KJWA==","E8XIR5mmV0AUXtJg6L9NPwLRb0RSOg==","2gt8yILRwRvJKFdDg9uLdAQAtc3ZTQ==","hpVmgoG5R2TwT01YBKt7qruAGBA9aQ==","8P7Sq+GFtgh7abbSaJBqqjifSxUkEw==","sjJPOO2bh/HYo75Z4itNjHN2SGFE2Q==","AYKgJ4ww4tG5W19pzNMQWdZ4bs6RnQ==","EppW78AZKo7Bdk4zpWgThdNNBYP0oA==","dyiExorC2FpfEFm1CWiQYtTM4t9ljA==","0A8SMQL42LBGZp0mm+YxyRCm/zucgQ==","taNhzuig2Qc+yfKXX8kU0hjROrxZDA==","CLZui2p0hZUqCtKwtmisQdmEb7YExg==","U2clFlTeHNphF/Idw1Z80vISvsYQ5Q==","jDozF2CQtNfwfUz2XxfuIvOsEZ7TFw==","16CteaC8z5dfrAnFteo+sEhohuwv/Q==","IDb191xETYiPPenqqpyjswGNp1cQRg==","WA70iCoSNz248T3TcdZk4cIicxaeGw==","u07Gp3OsSzua09DsrFdOzVccYQlagw==","cgAPPR0HpfoiITlvRwqokq2WmHLSzg==","3gBPDXNuhp521+CFM/eA59R6pVDNdg==","C/s9DxFVBW8e7+jqrfJTDPQUuBFI1w==","LXZrBug+OP2bI+R7R81T6SmyMej9jw==","0e4bdAM9DxSnrZerbBo5d8R6Ez118Q==","Jkmq3yvcCWw1JcFrwAZjxase8xsutA==","pZ14mci5Nlvyyx7gyYDW25eeOIGTtw==","UhlJ6FnakAX3pSkgAFoXUQIZlH+tHw==","a8TUC+5h5Zu3+WzVrxVlE2VJdZiDKg==","pxdxsz8g5y9xLdJd5CyRKk+t3eMZlg=="];

    // Crypto constants
    const _salt = 'W0rdl3S1x_2024!';

    // Private state
    let _w = null;
    let _g = false;
    let _ready = false;

    // Convert base64 to ArrayBuffer
    function b64ToAb(b64) {
        const bin = atob(b64);
        const buf = new Uint8Array(bin.length);
        for (let i = 0; i < bin.length; i++) buf[i] = bin.charCodeAt(i);
        return buf.buffer;
    }

    // Derive IV from index (deterministic)
    async function deriveIv(idx) {
        const data = new TextEncoder().encode('iv_' + idx.toString());
        const hash = await crypto.subtle.digest('SHA-256', data);
        return new Uint8Array(hash).slice(0, 12);
    }

    // Derive key using PBKDF2
    async function deriveKey(idx) {
        const keyMaterial = await crypto.subtle.importKey(
            'raw',
            new TextEncoder().encode(_salt + idx.toString()),
            'PBKDF2',
            false,
            ['deriveBits', 'deriveKey']
        );

        return crypto.subtle.deriveKey(
            {
                name: 'PBKDF2',
                salt: new TextEncoder().encode('pepper' + idx),
                iterations: 1000,
                hash: 'SHA-256'
            },
            keyMaterial,
            { name: 'AES-GCM', length: 256 },
            false,
            ['decrypt']
        );
    }

    // Decrypt a single word
    async function decryptWord(idx) {
        const encrypted = b64ToAb(_e[idx]);
        const tag = new Uint8Array(encrypted.slice(0, 16));
        const ciphertext = new Uint8Array(encrypted.slice(16));

        const iv = await deriveIv(idx);
        const key = await deriveKey(idx);

        // Combine ciphertext + tag for Web Crypto API
        const combined = new Uint8Array(ciphertext.length + 16);
        combined.set(ciphertext);
        combined.set(tag, ciphertext.length);

        const decrypted = await crypto.subtle.decrypt(
            { name: 'AES-GCM', iv: iv },
            key,
            combined
        );

        return new TextDecoder().decode(decrypted);
    }

    // Seeded PRNG
    function seededRandom(s) {
        const x = Math.sin(s) * 10000;
        return x - Math.floor(x);
    }

    // Get daily index
    function getDailyIndex() {
        const now = new Date();
        const start = new Date(2024, 0, 1);
        const days = Math.floor((now - start) / 86400000);
        return Math.floor(seededRandom(days + 12345) * _e.length);
    }

    // Initialize - decrypt today's word
    async function init() {
        const idx = getDailyIndex();
        _w = await decryptWord(idx);
        _ready = true;
    }

    // Start initialization
    const _initPromise = init();

    // Public API
    return {
        // Wait for initialization
        ready: function() { return _initPromise; },
        isReady: function() { return _ready; },

        // Check if guess wins
        checkWin: function(guess) {
            return _ready && guess === _w;
        },

        // Get letter feedback
        getLetterFeedback: function(guess) {
            if (!_ready) return Array(6).fill('absent');

            const result = Array(6).fill('absent');
            const target = _w.split('');
            const letters = guess.split('');

            for (let i = 0; i < 6; i++) {
                if (letters[i] === target[i]) {
                    result[i] = 'correct';
                    target[i] = null;
                }
            }

            for (let i = 0; i < 6; i++) {
                if (result[i] === 'correct') continue;
                const idx = target.indexOf(letters[i]);
                if (idx !== -1) {
                    result[i] = 'present';
                    target[idx] = null;
                }
            }

            return result;
        },

        // Reveal word (only after game over)
        revealWord: function() {
            return _g && _ready ? _w : '??????';
        },

        setGameOver: function() { _g = true; },
        isGameOver: function() { return _g; },

        reset: function() {
            _g = false;
            return init();
        },

        getDateString: function() {
            return new Date().toISOString().split('T')[0];
        },

        getTimeUntilNextWord: function() {
            const now = new Date();
            const tomorrow = new Date(now);
            tomorrow.setDate(tomorrow.getDate() + 1);
            tomorrow.setHours(0, 0, 0, 0);
            return tomorrow - now;
        },

        formatTime: function(ms) {
            const h = Math.floor(ms / 3600000);
            const m = Math.floor((ms % 3600000) / 60000);
            const s = Math.floor((ms % 60000) / 1000);
            return `${h.toString().padStart(2,'0')}:${m.toString().padStart(2,'0')}:${s.toString().padStart(2,'0')}`;
        }
    };
})();

// Legacy compatibility
function getDateString() { return WordGame.getDateString(); }
function getTimeUntilNextWord() { return WordGame.getTimeUntilNextWord(); }
function formatTime(ms) { return WordGame.formatTime(ms); }
