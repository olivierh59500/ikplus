# Keep reproducible build products inside the checkout.
export GOCACHE := $(CURDIR)/.cache/go-build

.PHONY: run build test vet verify autoplay assets verify-assets android android-run touch

run:
	go run .

build:
	mkdir -p bin
	go build -o bin/ikplus .

test:
	go test ./...

vet:
	go vet ./...

verify:
	go run ./cmd/verify -seed 1988 -round 13 -out .cache/verification/campaign.json

autoplay:
	go run . -autoplay -seed 1988

assets:
	python3 tools/extract_adf.py
	python3 tools/extract_shadow_projection.py
	python3 tools/extract_native_random.py
	python3 tools/extract_native_audio.py
	python3 tools/extract_native_graphics.py
	python3 tools/extract_native_gameplay.py
	gofmt -w internal/presentation/shadow_projection_data.go

verify-assets:
	python3 tools/extract_native_random.py --check
	python3 tools/extract_native_audio.py --check
	python3 tools/extract_native_graphics.py --check
	python3 tools/extract_native_gameplay.py --check

android:
	./scripts/build-android.sh

android-run:
	./scripts/run-android.sh

touch:
	go run . -touch

