mod parser;

fn main() {
    println!("Hello, world!");

    let result = parser::do_parse();

    match result {
        Ok(res) => println!("{:?}", res),
        Err(err) => println!("{}", err),
    }


}
