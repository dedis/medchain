clean:
	rm -rf build/co1 build/co2 build/co3 build/*.db build/public.toml build/*.cfg

spawn:
	go build
	bcadmin --config build create build/public.toml | tail -n 1