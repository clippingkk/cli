use inquire::{validator::StringValidator, Text, Password};
use std::error::Error;

pub(crate) struct AuthInPrompt {
	pub email: String,
	pub password: String
}

pub(crate) fn get_auth_from_prompt() -> Result<AuthInPrompt, Box<dyn Error>> {
	let validator: StringValidator = &|input| {
		let chars_count = input.chars().count();
		// TODO: check email
		if chars_count > 64 || chars_count < 6 {
			Err(String::from("email please~"))
		} else {
			Ok(())
		}
	};
	let pwd_validator: StringValidator = &|input| {
		let chars_count = input.chars().count();
		// TODO: check email
		if chars_count > 64 || chars_count < 4 {
			Err(String::from("password please~"))
		} else {
			Ok(())
		}
	};

	let email = Text::new("What's your email in `clippingkk` ?")
		.with_validator(validator)
		.prompt()?;

	let password = Password::new("What's your password that pair with email your previous provided ?")
		.with_validator(pwd_validator)
		.prompt()?;

	Ok(AuthInPrompt{
		email: email,
		password: password
	})
}
