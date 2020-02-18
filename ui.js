'use strict';

String.prototype.toTitleCase = function() {
	return this.split( ' ' )
		   .map( word => `${word.charAt( 0 ).toUpperCase()}${word.slice( 1 )}` )
		   .join( ' ' );
}
Array.prototype.last = function() {
	return this[ this.length - 1 ];
}
Array.prototype.unique = function() {
	return this.filter( ( x, i ) => this.indexOf( x ) === i );
}
Map.prototype.sortedKeys = function() {
	const keys = Array.from( this.keys() );
	keys.sort();
	return keys;
}
Map.prototype.getOrDefault = function( key, defaultValue ) {
	return this.has( key ) ? this.get( key ) : defaultValue;
}

const nameForID = id => id.split( '/' ).last()
			  .split( '_' )[0] 
			  .replace( '-', ' ' );

export class UI {
	constructor( name, sendMessageFunction ) {
		this.name = name;
		this.sendMessage = sendMessageFunction;

		this.values = new Map();
		this.updaters = new Map();
	}

	set( topic, value ) {
		this.values.set( topic, value );
	}

	get rooms() {
		return this.values.sortedKeys()
		                  .filter( k => k.startsWith( `${this.name}/` ) )
		                  .map( k => k.split( '/' )[ 1 ] )
		                  .unique();
	}
	devices( room ) {
		return this.values.sortedKeys()
		                  .filter( k => k.startsWith( `${room}/` ) )
		                  .map( k => k.split( '/' )[ 2 ] )
		                  .unique();
	}
	controls( device ) {
		return this.values.sortedKeys()
		                  .filter( k => k.startsWith( `${device}/` ) )
		                  .map( k => k.split( '/' )[ 3 ] )
		                  .unique();
	}

	render() {
		const _power = id => _div(
			_label( { for: id }, 'power' ),
			_input( {
				id: id,
				type: 'checkbox',
				checked: this.values.get( id ) === 'on' ? true : null,
				change: e => this.sendMessage(
					e.target.id,
					e.target.checked ? 'on' : 'off',
				),
			} ),
		);

		const _range = ( id, min, max ) => _div(
			_label( { for: id }, nameForID( id ) ),
			_input( {
				id: id,
				type: 'range',
				min: min,
				max: max,
				value: this.values.get( id ),
				change: e => sendMessage( e.target.id, e.target.value ),
			} ),
		);
		const _percent = id => _range( id, 0,    100  );
		const _degrees = id => _range( id, 0,    359  );
		const _kelvin  = id => _range( id, 2500, 9000 );

		const _text = id => _div(
			_label( { for: id }, nameForID( id ),),
			_input( {
				id: id,
				type: 'text',
				value: this.values.get( id ),
				change: e => sendMessage(
					e.target.id,
					e.target.value,
				),
			} ),
		);

		const _enum = id => _div(
			_label( { for: id }, nameForID( id ) ),
			_select(
				{ id: id },
				{ change: e => sendMessage( e.target.id, e.target.value ) },
				this.values.getOrDefault( `${id}/values`, '' )
				           .split( '\n' ).map( value => _option(
						value,
						{ selected: value === m.get( id ) ? true : null },
				) ),
			),
		);

		const _sensor = id => {
			const unit =
				id.endsWith( 'celsius' ) ? '°C' :
				id.endsWith( 'percent' ) ? '%'  :
							   ''   ;
			return _div(
				`${nameForID( id )}: `,
				_span( { id: id }, m.get( id ) ),
				unit,
			);
		};

		const _control = id =>
			id.includes( 'hygrothermograph' ) ? _sensor( id ) :
			id.endsWith( 'degrees' ) ? _degrees( id ) :
			id.endsWith( 'enum' )    ? _enum( id )    :
			id.endsWith( 'kelvin' )  ? _kelvin( id )  :
			id.endsWith( 'percent' ) ? _percent( id ) :
			id.endsWith( 'power' )   ? _power( id )   :
						   _text( id )    ;

		const _device = id => _section(
			{ id: id },
			_h3( nameForID( id ) ),
			this.controls( id ).map( control => _control( `${id}/${control}` ) ),
		);
		const _room = id => _section(
			{ id: id },
			_h2( nameForID( id ) ),
			this.devices( id ).map( device => _device( `${id}/${device}` ) ),
		);
		const main = _main(
			{ id: this.name },
			this.rooms.map( room => _room( `home/${room}` ) ),
		);

		document.getElementById( this.name ).replaceWith( main );
	}
}
