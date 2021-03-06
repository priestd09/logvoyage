class WSocket
	constructor: (@apiKey) ->
		@ws = new WebSocket("ws://" + window.location.hostname + ":12345/ws")
		@ws.onopen = (=> this.register())
		@ws.onmessage = (=> this.onMessage(event))

	register: ->
		@ws.send @apiKey
		console.log "registered user " + @apiKey

	onMessage: (event) ->
		data = JSON.parse event.data
		PubSub.publish data.type, data

$ ->
	new WSocket(options.apiKey)
