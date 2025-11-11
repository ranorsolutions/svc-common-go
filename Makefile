run:
	@command . ./.env && air

use_dev:
	@command go mod edit -replace github.com/ranorsolutions/http-common-go=../../../assets/lib/http-common-go

use_prod:
	@command go mod edit -dropreplace=github.com/ranorsolutions/http-common-go
