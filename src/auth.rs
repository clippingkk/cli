use inquire::{
    validator::{StringValidator, Validation},
    Password, Text,
};
use std::error::Error;

pub(crate) struct AuthInPrompt {
    pub email: String,
    pub password: String,
}

pub(crate) fn get_auth_from_prompt() -> Result<AuthInPrompt, Box<dyn Error>> {
    let validator = |input: &str| {
        let chars_count = input.chars().count();
        // TODO: check email
        if chars_count > 64 || chars_count < 6 {
            Ok(Validation::Invalid(("email please~").into()))
        } else {
            Ok(Validation::Valid)
        }
    };
    let pwd_validator = |input: &str| {
        let chars_count = input.chars().count();
        if chars_count > 64 || chars_count < 4 {
            Ok(Validation::Invalid(("password please~").into()))
        } else {
            Ok(Validation::Valid)
        }
    };

    let email = Text::new("What's your email in `clippingkk` ?")
        .with_validator(validator)
        .prompt()?;

    let password =
        Password::new("What's your password that pair with email your previous provided ?")
            .with_validator(pwd_validator)
            .prompt()?;

    Ok(AuthInPrompt {
        email: email,
        password: password,
    })
}
